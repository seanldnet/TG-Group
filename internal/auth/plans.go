package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidPlan = errors.New("invalid plan")
	ErrPlanConflict = errors.New("plan code already exists")
	ErrPlanNotFound = errors.New("plan not found")
)

type Plan struct {
	ID string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Description string `json:"description"`
	PriceUSDT int64 `json:"price_usdt"`
	DurationDays int32 `json:"duration_days"`
	BotLimit int32 `json:"bot_limit"`
	GroupLimit int32 `json:"group_limit"`
	Features map[string]any `json:"features"`
	SortOrder int32 `json:"sort_order"`
	Enabled bool `json:"enabled"`
}

func validatePlan(code,name,description string,price int64,durationDays,botLimit,groupLimit,sortOrder int32) error {
	code=strings.TrimSpace(strings.ToLower(code))
	name=strings.TrimSpace(name)
	if len(code)<2 || len(code)>40 || len(name)==0 || len(name)>80 || len(description)>2000 || price<0 || durationDays<1 || durationDays>36500 || botLimit<0 || groupLimit<0 || sortOrder<0 {
		return ErrInvalidPlan
	}
	for _,r:=range code { if !(r>='a'&&r<='z') && !(r>='0'&&r<='9') && r!='_' && r!='-' { return ErrInvalidPlan } }
	return nil
}

const planColumns = `id::text,code,name,description,price_usdt,duration_days,bot_limit,group_limit,features,sort_order,enabled`

func scanPlan(row pgx.Row)(Plan,error) {
	var item Plan
	err:=row.Scan(&item.ID,&item.Code,&item.Name,&item.Description,&item.PriceUSDT,&item.DurationDays,&item.BotLimit,&item.GroupLimit,&item.Features,&item.SortOrder,&item.Enabled)
	if item.Features==nil { item.Features=map[string]any{} }
	return item,err
}

func (s *Service) ListPlans(ctx context.Context)([]Plan,error) {
	rows,err:=s.pool.Query(ctx,`SELECT `+planColumns+` FROM plans WHERE enabled=true ORDER BY sort_order,price_usdt,id`)
	if err!=nil{return nil,fmt.Errorf("list plans: %w",err)}
	defer rows.Close()
	items:=make([]Plan,0)
	for rows.Next(){ item,scanErr:=scanPlan(rows);if scanErr!=nil{return nil,fmt.Errorf("scan plan: %w",scanErr)};items=append(items,item) }
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate plans: %w",err)}
	return items,nil
}

func (s *Service) ListAdminPlans(ctx context.Context)([]Plan,error) {
	rows,err:=s.pool.Query(ctx,`SELECT `+planColumns+` FROM plans ORDER BY sort_order,price_usdt,id`)
	if err!=nil{return nil,fmt.Errorf("list admin plans: %w",err)}
	defer rows.Close()
	items:=make([]Plan,0)
	for rows.Next(){ item,scanErr:=scanPlan(rows);if scanErr!=nil{return nil,fmt.Errorf("scan admin plan: %w",scanErr)};items=append(items,item) }
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate admin plans: %w",err)}
	return items,nil
}

func (s *Service) CreateAdminPlan(ctx context.Context,actorID string,input Plan)(Plan,error) {
	input.Code=strings.TrimSpace(strings.ToLower(input.Code));input.Name=strings.TrimSpace(input.Name)
	if err:=validatePlan(input.Code,input.Name,input.Description,input.PriceUSDT,input.DurationDays,input.BotLimit,input.GroupLimit,input.SortOrder);err!=nil{return Plan{},err}
	if input.Features==nil{input.Features=map[string]any{}}
	raw,err:=json.Marshal(input.Features);if err!=nil{return Plan{},ErrInvalidPlan}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{});if err!=nil{return Plan{},fmt.Errorf("begin create plan: %w",err)}
	defer tx.Rollback(ctx)
	item,err:=scanPlan(tx.QueryRow(ctx,`INSERT INTO plans(code,name,description,price_usdt,duration_days,bot_limit,group_limit,features,sort_order,enabled)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10) RETURNING `+planColumns,
		input.Code,input.Name,input.Description,input.PriceUSDT,input.DurationDays,input.BotLimit,input.GroupLimit,string(raw),input.SortOrder,input.Enabled))
	if isUniqueViolation(err){return Plan{},ErrPlanConflict}
	if err!=nil{return Plan{},fmt.Errorf("insert plan: %w",err)}
	var tenantID string
	if err:=tx.QueryRow(ctx,`SELECT tenant_id::text FROM users WHERE id=$1`,actorID).Scan(&tenantID);err!=nil{return Plan{},fmt.Errorf("load plan actor tenant: %w",err)}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,'plan.create','plan',$3)`,tenantID,actorID,item.ID);err!=nil{return Plan{},fmt.Errorf("audit plan create: %w",err)}
	if err:=tx.Commit(ctx);err!=nil{return Plan{},fmt.Errorf("commit plan create: %w",err)}
	return item,nil
}

func (s *Service) UpdateAdminPlan(ctx context.Context,actorID,planID string,input Plan)(Plan,error) {
	input.Name=strings.TrimSpace(input.Name)
	if len(input.Name)==0 || len(input.Name)>80 || len(input.Description)>2000 || input.PriceUSDT<0 || input.DurationDays<1 || input.DurationDays>36500 || input.BotLimit<0 || input.GroupLimit<0 || input.SortOrder<0{return Plan{},ErrInvalidPlan}
	if input.Features==nil{input.Features=map[string]any{}}
	raw,err:=json.Marshal(input.Features);if err!=nil{return Plan{},ErrInvalidPlan}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{});if err!=nil{return Plan{},fmt.Errorf("begin update plan: %w",err)}
	defer tx.Rollback(ctx)
	item,err:=scanPlan(tx.QueryRow(ctx,`UPDATE plans SET name=$2,description=$3,price_usdt=$4,duration_days=$5,bot_limit=$6,group_limit=$7,features=$8::jsonb,sort_order=$9,enabled=$10,updated_at=now()
		WHERE id=$1 RETURNING `+planColumns,planID,input.Name,input.Description,input.PriceUSDT,input.DurationDays,input.BotLimit,input.GroupLimit,string(raw),input.SortOrder,input.Enabled))
	if errors.Is(err,pgx.ErrNoRows){return Plan{},ErrPlanNotFound}
	if err!=nil{return Plan{},fmt.Errorf("update plan: %w",err)}
	var tenantID string
	if err:=tx.QueryRow(ctx,`SELECT tenant_id::text FROM users WHERE id=$1`,actorID).Scan(&tenantID);err!=nil{return Plan{},fmt.Errorf("load plan actor tenant: %w",err)}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,'plan.update','plan',$3)`,tenantID,actorID,item.ID);err!=nil{return Plan{},fmt.Errorf("audit plan update: %w",err)}
	if err:=tx.Commit(ctx);err!=nil{return Plan{},fmt.Errorf("commit plan update: %w",err)}
	return item,nil
}
