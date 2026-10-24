<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NButtonGroup, NPopover, NSpace } from 'naive-ui'
import { SmilePlus } from '@lucide/vue'
import ReactionUsers from '@/components/ReactionUsers.vue'
import type { Reaction } from '@/lib/api'
import { reactionEmoji, reactionKeys } from '@/lib/utils'
const props = defineProps<{
    commentId: number
    reactions: Reaction[]
    busy: boolean
  }>(),
  emit = defineEmits<{ reaction: [emoji: string] }>()
const selected = ref<string | null>(null),
  picker = ref(false)
const active = computed(() =>
  props.reactions.filter((reaction) => reaction.count > 0),
)
function react(emoji: string) {
  selected.value = null
  picker.value = false
  emit('reaction', emoji)
}
</script>
<template>
  <NSpace :size="6" align="center"
    ><NButtonGroup v-for="reaction in active" :key="reaction.emoji" size="tiny"
      ><NButton
        :type="reaction.reactedByMe ? 'primary' : 'default'"
        :secondary="reaction.reactedByMe"
        :disabled="busy"
        :aria-label="reactionEmoji[reaction.emoji]?.label"
        :aria-pressed="reaction.reactedByMe"
        @click="react(reaction.emoji)"
        >{{ reactionEmoji[reaction.emoji]?.symbol ?? reaction.emoji }}</NButton
      ><NPopover
        trigger="click"
        :show="selected === reaction.emoji"
        placement="bottom-start"
        :header-style="{ padding: '6px 12px' }"
        :content-style="{ padding: '4px 12px' }"
        @update:show="(show) => (selected = show ? reaction.emoji : null)"
        ><template #trigger
          ><NButton
            :type="reaction.reactedByMe ? 'primary' : 'default'"
            :secondary="reaction.reactedByMe"
            :aria-label="`查看${reactionEmoji[reaction.emoji]?.label}的人员，${reaction.count} 人`"
            :aria-expanded="selected === reaction.emoji"
            >{{ reaction.count }}</NButton
          ></template
        ><template #header
          >{{ reactionEmoji[reaction.emoji]?.label }}的人员</template
        ><ReactionUsers
          v-if="selected === reaction.emoji"
          :key="reaction.emoji"
          :comment-id="commentId"
          :emoji="reaction.emoji" /></NPopover></NButtonGroup
    ><NPopover v-model:show="picker" trigger="click" placement="bottom-start"
      ><template #trigger
        ><NButton
          quaternary
          circle
          size="tiny"
          :disabled="busy"
          aria-label="添加表情"
          :aria-expanded="picker"
          ><SmilePlus :size="14" /></NButton
      ></template>
      <div class="emoji-picker">
        <NButton
          v-for="emoji in reactionKeys"
          :key="emoji"
          class="emoji-picker__option"
          quaternary
          :aria-label="reactionEmoji[emoji]?.label"
          @click="react(emoji)"
          >{{ reactionEmoji[emoji]?.symbol }}</NButton
        >
      </div></NPopover
    ></NSpace
  >
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
