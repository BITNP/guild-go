<script setup lang="ts">
const { isDark } = useTheme();
const naiveTheme = computed(() => (isDark.value ? darkTheme : null));
const themeOverrides = computed(() =>
  isDark.value ? darkThemeOverrides : lightThemeOverrides,
);
</script>

<template>
  <NConfigProvider
    :date-locale="dateZhCN"
    :locale="zhCN"
    :theme="naiveTheme"
    :theme-overrides
  >
    <NGlobalStyle />
    <RouterView />
  </NConfigProvider>
</template>

<style scoped>
:global(:root) {
  --guild-primary: #1f5fbf;
  --guild-muted: #5e6b7b;
  --guild-border: #e1e6ec;
  --guild-surface: #fff;
  --guild-surface-subtle: #f7f9fc;
  --guild-text: #17202b;
  --guild-page-title-size: 28px;
  --guild-section-size: 18px;
  --guild-item-title-size: 15px;
  --guild-meta-size: 12px;
  --guild-danger: #c23b3b;
  --guild-success: #157347;
  --guild-warning: #966200;
  --guild-primary-soft: #e8f0ff;
  --guild-app-bg: #f3f5f8;
  --guild-header-bg: rgba(255, 255, 255, 0.86);
  --guild-shadow: rgba(23, 32, 43, 0.12);
}
:global([data-theme="dark"]) {
  --guild-primary: #76aaff;
  --guild-muted: #8797a6;
  --guild-border: #2b3946;
  --guild-surface: #17212b;
  --guild-surface-subtle: #1b2631;
  --guild-text: #e8eef4;
  --guild-danger: #f08b86;
  --guild-success: #56c99a;
  --guild-warning: #e6b85e;
  --guild-primary-soft: #1e3149;
  --guild-app-bg: #10161d;
  --guild-header-bg: rgba(19, 27, 35, 0.88);
  --guild-shadow: rgba(0, 0, 0, 0.36);
}
:global(html),
:global(body),
:global(#app) {
  margin: 0;
  min-height: 100%;
  width: 100%;
  background: var(--guild-app-bg);
  color: var(--guild-text);
}
:global(*) {
  box-sizing: border-box;
}
:global(a:focus-visible),
:global(:where(button, summary, [role="button"]):focus-visible) {
  outline: 2px solid var(--guild-primary);
  outline-offset: 3px;
}
@media (prefers-reduced-motion: reduce) {
  :global(*),
  :global(*::before),
  :global(*::after) {
    animation: none !important;
    transition: none !important;
    scroll-behavior: auto !important;
  }
}
</style>
