<script setup lang="ts">
import type { UserSummary } from "@/api";

const props = defineProps<{ commentId: number; emoji: string }>();
const people = ref<UserSummary[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

async function load() {
  loading.value = true;
  error.value = null;
  try {
    people.value = await fetchReactionUsers(props.commentId, props.emoji);
  } catch (err) {
    error.value = errorMessage(err, "无法加载人员列表");
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="reaction-users">
    <NSkeleton v-if="loading" :repeat="3" text />
    <NAlert v-else-if="error" role="alert" type="error">
      {{ error }}
      <NButton text @click="load">重新加载</NButton>
    </NAlert>
    <NEmpty
      v-else-if="people.length === 0"
      description="还没有人使用这个表情"
      size="small"
    />
    <NScrollbar v-else class="reaction-users__scroll">
      <NList>
        <NListItem
          v-for="person in people"
          :key="person.subject"
          class="reaction-person"
        >
          <UserIdentity
            class="reaction-person__identity"
            :size="24"
            :user="person"
          />
        </NListItem>
      </NList>
    </NScrollbar>
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
.reaction-users__scroll {
  max-height: 280px;
}
</style>
