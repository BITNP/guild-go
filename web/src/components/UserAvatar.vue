<script setup lang="ts">
const props = withDefaults(
  defineProps<{ name: string; src?: string | null; size?: number }>(),
  { size: 26 },
);
const initial = computed(() => props.name.slice(0, 1));
</script>

<template>
  <NAvatar
    color="#1f5fbf"
    :img-props="{ alt: '' }"
    object-fit="cover"
    round
    :size
    :src="src || undefined"
    :style="{ fontSize: `${Math.max(12, Math.round(size * 0.42))}px` }"
  >
    <template v-if="!src" #default>{{ initial }}</template>
    <template #fallback>
      <span class="avatar-initial">{{ initial }}</span>
    </template>
  </NAvatar>
</template>

<style scoped>
.avatar-initial {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  line-height: 1.25;
}
</style>
