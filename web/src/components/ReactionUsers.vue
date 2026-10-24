<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NEmpty,
  NList,
  NListItem,
  NScrollbar,
  NSkeleton,
} from 'naive-ui'
import UserIdentity from '@/components/UserIdentity.vue'
import { fetchReactionUsers, type UserSummary } from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = defineProps<{ commentId: number; emoji: string }>()
const people = ref<UserSummary[]>([]),
  loading = ref(true),
  error = ref<string | null>(null)
async function load() {
  loading.value = true
  error.value = null
  try {
    people.value = await fetchReactionUsers(props.commentId, props.emoji)
  } catch (err) {
    error.value = errorMessage(err, '无法加载人员列表')
  } finally {
    loading.value = false
  }
}
onMounted(load)
</script>
<template>
  <div class="reaction-users">
    <NSkeleton v-if="loading" text :repeat="3" /><NAlert
      v-else-if="error"
      type="error"
      role="alert"
      >{{ error }} <NButton text @click="load">重新加载</NButton></NAlert
    ><NEmpty
      v-else-if="!people.length"
      size="small"
      description="还没有人使用这个表情"
    /><NScrollbar v-else style="max-height: 280px"
      ><NList
        ><NListItem
          v-for="person in people"
          :key="person.subject"
          class="reaction-person"
          ><UserIdentity
            class="reaction-person__identity"
            :user="person"
            :size="24" /></NListItem></NList
    ></NScrollbar>
  </div>
</template>
<style scoped>
.reaction-users {
  width: 220px;
  max-width: calc(100vw - 64px);
}
.reaction-person {
  padding: 6px 0;
}
.reaction-person__identity {
  display: flex;
}
</style>
