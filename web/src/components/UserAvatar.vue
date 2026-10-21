<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { initials } from '@/lib/utils'

const props = withDefaults(defineProps<{
  name: string
  avatar?: string | null
  sizeClass?: string
}>(), {
  avatar: null,
  sizeClass: 'h-6 w-6 text-[10px]',
})

const failed = ref(false)
watch(() => props.avatar, () => { failed.value = false })
const initial = computed(() => initials(props.name))
</script>

<template>
  <img
    v-if="avatar && !failed"
    :src="avatar"
    :alt="name"
    :title="name"
    class="shrink-0 rounded-full border border-line/60 object-cover"
    :class="sizeClass"
    @error="failed = true"
  >
  <span
    v-else
    class="flex shrink-0 items-center justify-center rounded-full bg-accent-soft font-semibold text-accent"
    :class="sizeClass"
    :title="name"
  >
    {{ initial }}
  </span>
</template>