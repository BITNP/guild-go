<script setup lang="ts">
import type { Reaction } from "@/api";

const emit = defineEmits<{ reaction: [emoji: string] }>();
const props = defineProps<{
  commentId: number;
  reactions: Reaction[];
  busy: boolean;
}>();
const selected = ref<string | null>(null);
const picker = ref(false);
const active = computed(() =>
  props.reactions.filter((reaction) => reaction.count > 0),
);

function react(emoji: string) {
  selected.value = null;
  picker.value = false;
  emit("reaction", emoji);
}
</script>

<template>
  <NSpace align="center" :size="6">
    <NButtonGroup v-for="reaction in active" :key="reaction.emoji" size="tiny">
      <NButton
        :aria-label="reactionEmoji[reaction.emoji]?.label"
        :aria-pressed="reaction.reactedByMe"
        :disabled="busy"
        :secondary="reaction.reactedByMe"
        :type="reaction.reactedByMe ? 'primary' : 'default'"
        @click="react(reaction.emoji)"
      >
        {{ reactionEmoji[reaction.emoji]?.symbol ?? reaction.emoji }}
      </NButton>
      <NPopover
        :content-style="{ padding: '4px 12px' }"
        :header-style="{ padding: '6px 12px' }"
        placement="bottom-start"
        :show="selected === reaction.emoji"
        trigger="click"
        @update:show="(show) => (selected = show ? reaction.emoji : null)"
      >
        <template #trigger>
          <NButton
            :aria-expanded="selected === reaction.emoji"
            :aria-label="`查看${reactionEmoji[reaction.emoji]?.label}的人员，${reaction.count} 人`"
            :secondary="reaction.reactedByMe"
            :type="reaction.reactedByMe ? 'primary' : 'default'"
          >
            {{ reaction.count }}
          </NButton>
        </template>
        <template #header>
          {{ reactionEmoji[reaction.emoji]?.label }}的人员
        </template>
        <ReactionUsers
          v-if="selected === reaction.emoji"
          :key="reaction.emoji"
          :comment-id
          :emoji="reaction.emoji"
        />
      </NPopover>
    </NButtonGroup>
    <NPopover v-model:show="picker" placement="bottom-start" trigger="click">
      <template #trigger>
        <NButton
          :aria-expanded="picker"
          aria-label="添加表情"
          circle
          :disabled="busy"
          quaternary
          size="tiny"
        >
          <SmilePlus :size="14" />
        </NButton>
      </template>
      <div class="emoji-picker">
        <NButton
          v-for="emoji in reactionKeys"
          :key="emoji"
          :aria-label="reactionEmoji[emoji]?.label"
          class="emoji-picker__option"
          quaternary
          @click="react(emoji)"
        >
          {{ reactionEmoji[emoji]?.symbol }}
        </NButton>
      </div>
    </NPopover>
  </NSpace>
</template>

<style scoped>
.emoji-picker {
  display: grid;
  grid-template-columns: repeat(4, 36px);
  gap: 4px;
}
.emoji-picker__option {
  width: 36px;
  height: 36px;
  padding: 0;
  font-size: 20px;
}
</style>
