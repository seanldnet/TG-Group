<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';

const props = defineProps<{
  endpoint: string;
  mode: 'checkbox' | 'floating' | 'invisible';
}>();

const emit = defineEmits<{
  solved: [token: string];
  error: [message: string];
}>();

const widgetHost = ref<HTMLElement>();
const token = ref('');
let widgetElement: HTMLElement | null = null;
let disposed = false;

const widgetModuleURL =
  'https://cdn.jsdelivr.net/npm/@cap.js/widget@0.1.63';

async function loadModule(url: string) {
  return import(/* @vite-ignore */ url);
}

function handleSolve(event: Event) {
  const detail = (event as CustomEvent<{ token?: string }>).detail;
  token.value = detail?.token ?? '';
  if (token.value) emit('solved', token.value);
}

function handleReset() {
  token.value = '';
  emit('solved', '');
}

function handleError(event: Event) {
  const detail = (event as CustomEvent<{ message?: string }>).detail;
  emit('error', detail?.message || 'Cap 人机验证暂时不可用，请稍后重试');
}

onMounted(async () => {
  try {
    const capModule = await loadModule(widgetModuleURL);
    if (disposed) return;

    if (props.mode === 'invisible') return;

    widgetElement = document.createElement('cap-widget');
    widgetElement.id = 'tg-group-cap-login-widget';
    widgetElement.setAttribute('data-cap-api-endpoint', props.endpoint);
    widgetElement.setAttribute('data-cap-lang', 'zh');
    widgetElement.setAttribute('data-cap-i18n-initial-state', '点击进行安全验证');
    widgetElement.setAttribute('data-cap-i18n-verifying-label', '正在验证...');
    widgetElement.setAttribute('data-cap-i18n-solved-label', '验证成功');
    widgetElement.setAttribute('data-cap-i18n-error-label', '验证失败，请重试');
    widgetElement.setAttribute('data-cap-disable-haptics', '');
    widgetElement.addEventListener('solve', handleSolve);
    widgetElement.addEventListener('reset', handleReset);
    widgetElement.addEventListener('error', handleError);
    widgetHost.value?.append(widgetElement);

    if (props.mode === 'floating') {
      await loadModule(
        'https://cdn.jsdelivr.net/npm/@cap.js/widget@0.1.63/floating.js',
      );
    }
    // Keep the constructor available for the invisible solve path.
    (window as Window & { __tgGroupCap?: unknown }).__tgGroupCap =
      capModule.default;
  } catch {
    emit('error', 'Cap 验证组件加载失败，请检查服务器是否可以访问 jsDelivr');
  }
});

async function getToken(): Promise<string> {
  if (props.mode === 'invisible') {
    try {
      const capModule = await loadModule(widgetModuleURL);
      const Cap = capModule.default;
      const solver = new Cap({ apiEndpoint: props.endpoint });
      const result = await solver.solve();
      const solvedToken = String(result?.token ?? '');
      if (!solvedToken) throw new Error('Cap 未返回验证令牌');
      token.value = solvedToken;
      emit('solved', solvedToken);
      return solvedToken;
    } catch (error) {
      emit(
        'error',
        error instanceof Error ? error.message : '隐形验证失败，请重试',
      );
      throw error;
    }
  }

  if (!token.value) {
    throw new Error(
      props.mode === 'floating'
        ? '请先点击安全验证并完成验证'
        : '请先完成安全验证',
    );
  }
  return token.value;
}

onBeforeUnmount(() => {
  disposed = true;
  widgetElement?.removeEventListener('solve', handleSolve);
  widgetElement?.removeEventListener('reset', handleReset);
  widgetElement?.removeEventListener('error', handleError);
  widgetElement?.remove();
});

defineExpose({ getToken });
</script>

<template>
  <div class="cap-login-verification w-full max-w-full overflow-hidden">
    <div
      v-if="mode !== 'invisible'"
      ref="widgetHost"
      class="cap-widget-host w-full max-w-full"
    />
    <button
      v-if="mode === 'floating'"
      type="button"
      data-cap-floating="#tg-group-cap-login-widget"
      data-cap-floating-position="bottom"
      class="cap-floating-trigger"
    >
      {{ token ? '验证成功' : '点击进行安全验证' }}
    </button>
    <p v-if="mode === 'invisible'" class="text-muted-foreground text-xs">
      登录时将自动进行安全验证
    </p>
  </div>
</template>

<style scoped>
.cap-login-verification {
  box-sizing: border-box;
}

.cap-widget-host :deep(cap-widget) {
  display: block;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  --cap-widget-width: 100%;
  --cap-widget-height: 40px;
  --cap-widget-border-radius: 6px;
  --cap-widget-padding: 8px;
  --cap-font: inherit;
}

.cap-floating-trigger {
  display: flex;
  width: 100%;
  max-width: 100%;
  min-height: 40px;
  align-items: center;
  justify-content: center;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 14px;
}

.cap-floating-trigger:hover {
  border-color: var(--ant-color-primary, #1677ff);
  color: var(--ant-color-primary, #1677ff);
}
</style>
