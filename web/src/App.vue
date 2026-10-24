<script setup lang="ts">
import { NConfigProvider, NGlobalStyle, zhCN, dateZhCN } from 'naive-ui'
import { theme } from '@/lib/theme'
</script>
<template>
  <NConfigProvider
    :locale="zhCN"
    :date-locale="dateZhCN"
    :theme-overrides="theme"
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
  --guild-danger: #c23b3b;
}
:global(html),
:global(body),
:global(#app) {
  margin: 0;
  min-height: 100%;
  width: 100%;
}
:global(*) {
  box-sizing: border-box;
}
:global(a:focus-visible) {
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
