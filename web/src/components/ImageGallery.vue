<script setup lang="ts">
import { cloneVNode } from 'vue'
import { NImageGroup, type ImageGroupRenderToolbar } from 'naive-ui'

const renderToolbar: ImageGroupRenderToolbar = ({ nodes }) => {
  const labels = {
    prev: '上一张图片',
    next: '下一张图片',
    rotateCounterclockwise: '向左旋转',
    rotateClockwise: '向右旋转',
    resizeToOriginalSize: '查看原始尺寸',
    zoomOut: '缩小图片',
    zoomIn: '放大图片',
    download: '下载图片',
    close: '关闭图片预览',
  }
  return (Object.keys(labels) as (keyof typeof labels)[]).map((key) => {
    const node = nodes[key]
    return cloneVNode(node, {
      role: 'button',
      tabindex: 0,
      'aria-label': labels[key],
      title: labels[key],
      class: 'guild-image-preview-tool',
      onKeydown: (event: KeyboardEvent) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          node.props?.onClick?.(event)
        }
      },
    })
  })
}
</script>

<template>
  <NImageGroup :render-toolbar="renderToolbar"><slot /></NImageGroup>
</template>

<style scoped>
:global(.guild-image-preview-tool:focus-visible) {
  outline: 2px solid white;
  outline-offset: 3px;
  border-radius: 4px;
}
</style>
