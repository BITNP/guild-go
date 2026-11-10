<script setup lang="ts">
defineProps<{ title: string }>();

const headingId = useId();
</script>

<template>
  <section :aria-labelledby="headingId" class="panel-section">
    <div class="section-heading">
      <h3 :id="headingId">{{ title }}</h3>
      <slot name="actions" />
    </div>
    <slot />
  </section>
</template>

<style scoped>
.panel-section {
  padding: 19px 24px;
  border-bottom: 1px solid rgba(139, 164, 169, 0.13);
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 13px;
}

.section-heading h3 {
  margin: 0;
  color: #e7c47b;
  font-size: 15px;
  font-weight: 700;
}

[data-theme="light"] .panel-section {
  border-bottom-color: rgba(82, 105, 116, 0.13);
}

[data-theme="light"] .section-heading h3 {
  color: #8e590d;
}

@media (max-width: 640px) {
  .panel-section {
    padding-right: 18px;
    padding-left: 18px;
  }
}
</style>
