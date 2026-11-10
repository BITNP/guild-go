<script setup lang="ts">
defineProps<{ tone?: "primary" | "danger" }>();
</script>

<template>
  <button class="panel-button" :class="tone" type="button"><slot /></button>
</template>

<style scoped>
.panel-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 9px;
  border: 1px solid rgba(221, 176, 75, 0.3);
  border-radius: 2px;
  background: transparent;
  color: #d9b96f;
  font-size: 13px;
  cursor: pointer;
}

.panel-button:hover:not(:disabled) {
  border-color: rgba(240, 187, 78, 0.68);
  color: #ffe0a2;
}

.panel-button:focus-visible {
  outline: 2px solid currentColor;
  outline-offset: 3px;
}

.panel-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.primary {
  border-color: rgba(232, 181, 75, 0.58);
  background: rgba(218, 170, 67, 0.1);
  color: #f0cb7e;
}

.danger {
  border-color: rgba(226, 125, 104, 0.34);
  color: #e3a091;
}

.danger:hover:not(:disabled) {
  border-color: rgba(241, 143, 119, 0.7);
  background: rgba(177, 69, 50, 0.12);
  color: #ffb9a8;
}

[data-theme="light"] .panel-button:not(.danger) {
  border-color: rgba(155, 101, 17, 0.3);
  background: rgba(184, 119, 21, 0.06);
  color: #80510d;
}

[data-theme="light"] .panel-button:hover:not(:disabled, .danger) {
  border-color: rgba(157, 100, 17, 0.48);
  color: #714707;
}

[data-theme="light"] .danger {
  border-color: rgba(163, 63, 54, 0.34);
  color: #a33f36;
}

[data-theme="light"] .danger:hover:not(:disabled) {
  border-color: rgba(163, 63, 54, 0.64);
  background: rgba(163, 63, 54, 0.08);
  color: #852e27;
}
</style>
