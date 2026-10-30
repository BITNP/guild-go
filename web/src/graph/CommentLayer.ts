import * as THREE from 'three'
import type { ProgressItem } from '@/lib/api'
import { reactionEmoji, reactionKeys } from '@/lib/utils'
import type { ThemeMode } from '@/composables/useTheme'

interface ProgressVisual {
  anchor: THREE.Group
  card: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>
  dot: THREE.Mesh<THREE.SphereGeometry, THREE.MeshBasicMaterial>
  line: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>
  textureKey: string
  y: number
  targetY: number
  opacity: number
  reveal: number
  pulse: number
  item: ProgressItem
}

export interface ProgressCardHit {
  key: string
  commentId: number | null
  emoji: string | null
}

export interface ComposerPose {
  position: THREE.Vector3
  rotationY: number
  opacity: number
}

export const COMMENT_CARD_WIDTH = 6.2
export const COMMENT_CARD_HEIGHT = 2.36

const CARD_WIDTH = COMMENT_CARD_WIDTH
const CARD_HEIGHT = COMMENT_CARD_HEIGHT
const CARD_CENTER_X = CARD_WIDTH / 2 + 0.28
const BASE_Y = 2.62
const SPACING = 2.85
const FADE_START = 6.2
const FADE_END = 15.5
const CARD_CANVAS_WIDTH = 1024
const CARD_CANVAS_HEIGHT = 400
const REACTION_ROW_TOP = 300
const REACTION_ROW_BOTTOM = 374
const REACTION_START_X = 34
const REACTION_CELL_WIDTH = 110
const REACTION_GAP = 10

const CARD_PALETTES = {
  dark: {
    surface: 'rgba(8, 14, 17, 0.9)',
    author: '#f3c35f',
    meta: 'rgba(178, 201, 208, 0.68)',
    body: 'rgba(240, 244, 238, 0.94)',
    accentSoft: 'rgba(239, 180, 66, 0.92)',
    cyan: 'rgba(102, 219, 186, 0.82)',
    divider: 'rgba(139, 166, 171, 0.2)',
    empty: 'rgba(232, 185, 85, 0.86)',
  },
  light: {
    surface: 'rgba(255, 255, 255, 0.94)',
    author: '#a56812',
    meta: 'rgba(69, 87, 101, 0.74)',
    body: 'rgba(23, 32, 43, 0.96)',
    accentSoft: 'rgba(181, 120, 24, 0.9)',
    cyan: 'rgba(21, 124, 89, 0.9)',
    divider: 'rgba(82, 105, 119, 0.18)',
    empty: 'rgba(133, 83, 12, 0.86)',
  },
} as const

export class CommentLayer {
  readonly group = new THREE.Group()
  readonly composerPose: ComposerPose = {
    position: new THREE.Vector3(),
    rotationY: 0,
    opacity: 0,
  }
  private readonly cardGeometry = new THREE.PlaneGeometry(CARD_WIDTH, CARD_HEIGHT)
  private readonly dotGeometry = new THREE.SphereGeometry(0.15, 16, 12)
  private readonly composerDot: THREE.Mesh<THREE.SphereGeometry, THREE.MeshBasicMaterial>
  private readonly composerLine: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>
  private readonly emptyLabel: THREE.Sprite
  private readonly visuals = new Map<string, ProgressVisual>()
  private targetScroll = 0
  private scroll = 0
  private lastNodePosition: THREE.Vector3 | null = null
  private commentsLoading = false
  private emptyEnabled = true
  private emptyOpacity = 0
  private composerVisible = false
  private composerOpacity = 0
  private composerY = BASE_Y
  private theme: ThemeMode = 'dark'

  constructor() {
    this.group.visible = false
    this.group.renderOrder = 5
    this.composerDot = new THREE.Mesh(
      this.dotGeometry,
      new THREE.MeshBasicMaterial({
        color: 0xe7b34d,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        blending: THREE.AdditiveBlending,
      }),
    )
    this.composerLine = new THREE.Line(
      new THREE.BufferGeometry().setFromPoints([
        new THREE.Vector3(0, 0, 0),
        new THREE.Vector3(0, 1, 0),
      ]),
      new THREE.LineBasicMaterial({
        color: 0xd8a84c,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
      }),
    )
    this.emptyLabel = new THREE.Sprite(new THREE.SpriteMaterial({
      map: createEmptyTexture(this.theme),
      transparent: true,
      opacity: 0,
      depthWrite: false,
      depthTest: false,
      toneMapped: false,
    }))
    this.emptyLabel.scale.set(6.2, 0.98, 1)
    this.emptyLabel.visible = false
    this.group.add(this.composerLine, this.composerDot, this.emptyLabel)
  }

  setProgress(items: ProgressItem[], loading = false) {
    this.commentsLoading = loading
    const previousCount = this.visuals.size
    const previousIDs = new Set(this.visuals.keys())
    const previousCommentIDs = [...previousIDs]
      .filter((key) => key.startsWith('comment:'))
      .map((key) => Number(key.slice('comment:'.length)))
    const previousMaxCommentID = previousCommentIDs.length
      ? Math.max(
          ...previousCommentIDs,
        )
      : Number.NEGATIVE_INFINITY
    const active = new Set(items.map(progressKey))
    for (const [id, visual] of this.visuals) {
      if (active.has(id)) continue
      this.group.remove(visual.anchor, visual.line)
      this.disposeVisual(visual)
      this.visuals.delete(id)
    }

    items.forEach((item, index) => {
      const id = progressKey(item)
      let visual = this.visuals.get(id)
      if (!visual) {
        visual = this.createVisual(item)
        this.visuals.set(id, visual)
        this.group.add(visual.anchor, visual.line)
      } else {
        visual.item = item
        const textureKey = progressTextureKey(item)
        if (visual.textureKey !== textureKey) {
          const previousTexture = visual.card.material.map
          visual.card.material.map = createProgressTexture(item, this.theme)
          visual.card.material.needsUpdate = true
          previousTexture?.dispose()
          visual.textureKey = textureKey
          visual.dot.material.color.setHex(
            item.kind === 'comment' && item.comment.reactions.length > 0
              ? 0x6de2c2
              : 0xe7b34d,
          )
          visual.pulse = 1
        }
      }
      visual.targetY = this.progressY(items.length, index)
    })

    if (previousCount > 0 && items.length > previousCount) {
      const added = (items.length - previousCount) * SPACING
      const hasNewProgress = items.some((item) =>
        item.kind === 'comment' &&
        !previousIDs.has(progressKey(item)) &&
        item.comment.id > previousMaxCommentID,
      )
      if (hasNewProgress) {
        this.targetScroll = 0
      } else {
        this.scroll -= added
        this.targetScroll -= added
      }
    }
    this.clampScroll()
  }

  setComposerVisible(visible: boolean) {
    if (this.composerVisible === visible) return
    this.composerVisible = visible
    const comments = [...this.visuals.values()]
    comments.forEach((visual, index) => {
      visual.targetY = this.progressY(comments.length, index)
    })
    if (visible) {
      this.scroll = 0
      this.targetScroll = 0
    }
    this.clampScroll()
  }

  setTheme(theme: ThemeMode) {
    this.theme = theme
    for (const visual of this.visuals.values()) {
      const previousTexture = visual.card.material.map
      visual.card.material.map = createProgressTexture(
        visual.item,
        theme,
      )
      visual.card.material.needsUpdate = true
      previousTexture?.dispose()
    }
    const previousEmpty = this.emptyLabel.material.map
    this.emptyLabel.material.map = createEmptyTexture(theme)
    this.emptyLabel.material.needsUpdate = true
    previousEmpty?.dispose()
  }

  setEmptyEnabled(enabled: boolean) {
    this.emptyEnabled = enabled
    if (!enabled) {
      this.emptyOpacity = 0
      this.emptyLabel.material.opacity = 0
      this.emptyLabel.visible = false
    }
  }

  scrollBy(delta: number): boolean {
    this.targetScroll += delta * 0.022
    this.clampScroll()
    return this.targetScroll <= this.minScroll() + 0.15
  }

  pickProgressAt(raycaster: THREE.Raycaster): ProgressCardHit | null {
    const cards = [...this.visuals.values()]
      .filter((visual) => visual.opacity > 0.04 && visual.card.visible)
      .map((visual) => visual.card)
    const hit = raycaster.intersectObjects(cards, false)[0]
    if (!hit?.uv) return null
    const key = hit.object.userData.progressKey
    const commentId = hit.object.userData.commentId
    if (typeof key !== 'string') return null
    const resolvedCommentID =
      typeof commentId === 'number' ? commentId : null
    const x = hit.uv.x * CARD_CANVAS_WIDTH
    const y = (1 - hit.uv.y) * CARD_CANVAS_HEIGHT
    if (
      resolvedCommentID == null ||
      y < REACTION_ROW_TOP ||
      y > REACTION_ROW_BOTTOM
    ) {
      return { key, commentId: resolvedCommentID, emoji: null }
    }
    const slotWidth = REACTION_CELL_WIDTH + REACTION_GAP
    const index = Math.floor((x - REACTION_START_X) / slotWidth)
    const withinSlot = (x - REACTION_START_X) % slotWidth <= REACTION_CELL_WIDTH
    if (
      index < 0 ||
      index >= reactionKeys.length ||
      !withinSlot
    ) {
      return { key, commentId: resolvedCommentID, emoji: null }
    }
    return {
      key,
      commentId: resolvedCommentID,
      emoji: reactionKeys[index] ?? null,
    }
  }

  update(
    delta: number,
    camera: THREE.Camera,
    nodeWorldPosition: THREE.Vector3,
    selected: boolean,
  ) {
    if (selected) {
      if (!this.lastNodePosition) this.lastNodePosition = new THREE.Vector3()
      this.lastNodePosition.copy(nodeWorldPosition)
    }
    if (!this.lastNodePosition) {
      this.group.visible = false
      return
    }
    const scrollDamping = 1 - Math.exp(-delta * 13)
    const positionDamping = 1 - Math.exp(-delta * 10)
    const opacityDamping = 1 - Math.exp(-delta * 9)
    const revealDamping = 1 - Math.exp(-delta * 11)
    this.scroll += (this.targetScroll - this.scroll) * scrollDamping
    const cameraRight = new THREE.Vector3(1, 0, 0).applyQuaternion(camera.quaternion)
    cameraRight.y = 0
    const cardRotation = cameraRight.lengthSq() > 0.0001
      ? Math.atan2(-cameraRight.z, cameraRight.x)
      : 0

    for (const visual of this.visuals.values()) {
      const y = visual.targetY + this.scroll
      visual.y += (y - visual.y) * positionDamping
      visual.anchor.position.set(this.lastNodePosition.x, this.lastNodePosition.y + y, this.lastNodePosition.z)
      visual.anchor.rotation.set(0, cardRotation, 0)
      const positions = visual.line.geometry.getAttribute('position') as THREE.BufferAttribute
      positions.setXYZ(0, this.lastNodePosition.x, this.lastNodePosition.y, this.lastNodePosition.z)
      positions.setXYZ(1, this.lastNodePosition.x, this.lastNodePosition.y + y, this.lastNodePosition.z)
      positions.needsUpdate = true

      const farFade = 1 - smoothstep(FADE_START, FADE_END, visual.y)
      const verticalVisible = smoothstep(0.75, 1.7, visual.y)
      const targetOpacity = selected ? farFade * verticalVisible : 0
      visual.opacity += (targetOpacity - visual.opacity) * opacityDamping
      visual.reveal += (1 - visual.reveal) * revealDamping
      visual.pulse *= Math.exp(-delta * 5.5)
      const revealScale = 0.93 + visual.reveal * 0.07
      const pulseScale = 1 + visual.pulse * 0.028
      visual.card.scale.setScalar(revealScale * pulseScale)
      visual.card.material.opacity = visual.opacity * visual.reveal * 0.9
      visual.dot.material.opacity = visual.opacity
      visual.line.material.opacity = visual.opacity * 0.3
      visual.card.visible = visual.opacity > 0.012 && visual.reveal > 0.01
      visual.dot.visible = visual.opacity > 0.012
      visual.line.visible = visual.opacity > 0.012
    }

    const composerTargetY = BASE_Y + this.scroll
    this.composerY += (composerTargetY - this.composerY) * positionDamping
    const composerTargetOpacity = selected && this.composerVisible
      ? (1 - smoothstep(FADE_START, FADE_END, this.composerY)) *
        smoothstep(0.75, 1.7, this.composerY)
      : 0
    this.composerOpacity +=
      (composerTargetOpacity - this.composerOpacity) * opacityDamping
    const composerAnchor = new THREE.Vector3(
      this.lastNodePosition.x,
      this.lastNodePosition.y + this.composerY,
      this.lastNodePosition.z,
    )
    const composerRight = cameraRight.lengthSq() > 0.0001
      ? cameraRight.clone().normalize()
      : new THREE.Vector3(1, 0, 0)
    this.composerPose.position
      .copy(composerAnchor)
      .addScaledVector(composerRight, CARD_CENTER_X)
    this.composerPose.rotationY = cardRotation
    this.composerPose.opacity = this.composerOpacity
    this.composerDot.position.copy(composerAnchor)
    this.composerDot.material.opacity = this.composerOpacity
    this.composerDot.visible = this.composerOpacity > 0.012
    const composerPositions =
      this.composerLine.geometry.getAttribute('position') as THREE.BufferAttribute
    composerPositions.setXYZ(
      0,
      this.lastNodePosition.x,
      this.lastNodePosition.y,
      this.lastNodePosition.z,
    )
    composerPositions.setXYZ(
      1,
      this.lastNodePosition.x,
      this.lastNodePosition.y + this.composerY,
      this.lastNodePosition.z,
    )
    composerPositions.needsUpdate = true
    this.composerLine.material.opacity = this.composerOpacity * 0.3
    this.composerLine.visible = this.composerOpacity > 0.012

    const showEmpty = selected &&
      this.emptyEnabled &&
      !this.composerVisible &&
      !this.commentsLoading &&
      this.visuals.size === 0
    this.emptyOpacity += ((showEmpty ? 1 : 0) - this.emptyOpacity) * opacityDamping
    this.emptyLabel.position.set(
      this.lastNodePosition.x,
      this.lastNodePosition.y + BASE_Y + 0.2,
      this.lastNodePosition.z,
    )
    this.emptyLabel.material.opacity = this.emptyOpacity
    this.emptyLabel.visible = this.emptyOpacity > 0.012
    this.group.visible = this.emptyOpacity > 0.012
      || this.composerOpacity > 0.012
      || [...this.visuals.values()].some((visual) => visual.opacity > 0.012)
  }

  dispose() {
    for (const visual of this.visuals.values()) this.disposeVisual(visual)
    this.visuals.clear()
    this.cardGeometry.dispose()
    this.dotGeometry.dispose()
    this.composerDot.material.dispose()
    this.composerLine.geometry.dispose()
    this.composerLine.material.dispose()
    this.emptyLabel.material.map?.dispose()
    this.emptyLabel.material.dispose()
    this.group.clear()
  }

  private createVisual(item: ProgressItem): ProgressVisual {
    const texture = createProgressTexture(item, this.theme)
    const card = new THREE.Mesh(
      this.cardGeometry,
      new THREE.MeshBasicMaterial({
        map: texture,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        side: THREE.DoubleSide,
      }),
    )
    card.userData.progressKey = progressKey(item)
    if (item.kind === 'comment') card.userData.commentId = item.comment.id
    const dot = new THREE.Mesh(
      this.dotGeometry,
      new THREE.MeshBasicMaterial({
        color: item.kind === 'comment' && item.comment.reactions.length > 0
          ? 0x6de2c2
          : 0xe7b34d,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        blending: THREE.AdditiveBlending,
      }),
    )
    const anchor = new THREE.Group()
    card.position.set(CARD_CENTER_X, 0, 0)
    dot.position.set(0, 0, 0)
    anchor.add(card, dot)
    const lineGeometry = new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(0, 0, 0),
      new THREE.Vector3(0, 1, 0),
    ])
    const line = new THREE.Line(
      lineGeometry,
      new THREE.LineBasicMaterial({
        color: 0xd8a84c,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        blending: THREE.NormalBlending,
      }),
    )
    return {
      anchor,
      card,
      dot,
      line,
      textureKey: progressTextureKey(item),
      y: BASE_Y,
      targetY: BASE_Y,
      opacity: 0,
      reveal: 0,
      pulse: 0,
      item,
    }
  }

  private disposeVisual(visual: ProgressVisual) {
    visual.card.material.map?.dispose()
    visual.card.material.dispose()
    visual.dot.material.dispose()
    visual.line.geometry.dispose()
    visual.line.material.dispose()
  }

  private progressY(count: number, index: number): number {
    const composerOffset = this.composerVisible ? 1 : 0
    return BASE_Y + (count - 1 - index + composerOffset) * SPACING
  }

  private minScroll(): number {
    if (this.visuals.size === 0) return 0
    const composerOffset = this.composerVisible ? 1 : 0
    return -(this.visuals.size - 1 + composerOffset) * SPACING
  }

  private clampScroll() {
    const minimum = this.minScroll()
    this.targetScroll = Math.max(minimum, Math.min(0, this.targetScroll))
    this.scroll = Math.max(minimum, Math.min(0, this.scroll))
  }
}

function createProgressTexture(
  item: ProgressItem,
  theme: ThemeMode,
): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = CARD_CANVAS_WIDTH
  canvas.height = CARD_CANVAS_HEIGHT
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context is unavailable')
  const palette = CARD_PALETTES[theme]

  context.clearRect(0, 0, canvas.width, canvas.height)
  context.fillStyle = palette.surface
  context.fillRect(0, 0, canvas.width, canvas.height)
  context.fillStyle = palette.accentSoft
  context.fillRect(0, 0, 6, canvas.height)
  context.fillStyle = palette.cyan
  context.fillRect(canvas.width - 4, 0, 4, canvas.height)

  context.font = '600 46px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = palette.author
  context.textBaseline = 'middle'
  const author = item.kind === 'comment'
    ? item.comment.author
    : item.descriptionChange.author
  const createdAt = item.kind === 'comment'
    ? item.comment.createdAt
    : item.descriptionChange.createdAt
  const authorLabel = item.kind === 'description_change'
    ? `${author.userName} · 描述更新`
    : author.userName
  context.fillText(trimToWidth(context, authorLabel, 590, 46), 34, 42)
  context.font = '34px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = palette.meta
  const time = new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(createdAt))
  context.textAlign = 'right'
  context.fillText(`${time} · 展开`, 990, 42)
  context.textAlign = 'left'

  context.font = '44px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = palette.body
  const comment = item.kind === 'comment' ? item.comment : null
  const change = item.kind === 'description_change'
    ? item.descriptionChange
    : null
  const rawBody = comment?.body ?? change?.after ?? ''
  const body = rawBody.trim()
    ? rawBody
    : comment
      ? comment.attachments.length > 0
        ? '图片进展'
        : '无正文'
      : '清空了节点描述'
  const wrapped = wrapTextPreserving(context, body, 940, 4)
  wrapped.lines.forEach((line, index) =>
    context.fillText(line, 34, 96 + index * 52),
  )
  if (comment && comment.attachments.length > 0) {
    context.font = '26px "Noto Sans SC", "PingFang SC", sans-serif'
    context.fillStyle = palette.cyan
    context.textAlign = 'right'
    context.fillText(`附 ${comment.attachments.length} 张图片`, 990, 276)
    context.textAlign = 'left'
  }
  context.strokeStyle = palette.accentSoft
  context.lineWidth = 1
  context.beginPath()
  context.moveTo(34, 76)
  context.lineTo(990, 76)
  context.stroke()
  context.strokeStyle = palette.divider
  context.beginPath()
  context.moveTo(34, 288)
  context.lineTo(990, 288)
  context.stroke()

  if (comment) {
    const counts = new Map(
      comment.reactions
        .filter((reaction) => reaction.count > 0)
        .map((reaction) => [reaction.emoji, reaction]),
    )
    reactionKeys.forEach((emoji, index) => {
      const x = REACTION_START_X + index * (REACTION_CELL_WIDTH + REACTION_GAP)
      const reaction = counts.get(emoji)
      context.fillStyle = reaction?.reactedByMe
        ? 'rgba(224, 174, 66, 0.27)'
        : reaction
          ? 'rgba(75, 201, 174, 0.15)'
          : 'rgba(91, 119, 124, 0.1)'
      context.fillRect(x, REACTION_ROW_TOP, REACTION_CELL_WIDTH, 72)
      context.strokeStyle = reaction?.reactedByMe
        ? 'rgba(241, 192, 87, 0.78)'
        : reaction
          ? 'rgba(98, 218, 186, 0.48)'
          : 'rgba(103, 137, 142, 0.24)'
      context.strokeRect(
        x + 0.5,
        REACTION_ROW_TOP + 0.5,
        REACTION_CELL_WIDTH - 1,
        71,
      )

      context.font = '32px "Apple Color Emoji", "Segoe UI Emoji", sans-serif'
      context.textAlign = 'center'
      context.textBaseline = 'middle'
      context.fillStyle = reaction
        ? 'rgba(244, 247, 239, 0.98)'
        : 'rgba(176, 196, 198, 0.48)'
      context.fillText(
        reactionEmoji[emoji]?.symbol ?? emoji,
        x + (reaction ? 40 : REACTION_CELL_WIDTH / 2),
        REACTION_ROW_TOP + 36,
      )
      if (reaction) {
        context.font = '600 25px "Noto Sans SC", "PingFang SC", sans-serif'
        context.fillStyle = reaction.reactedByMe
          ? 'rgba(255, 220, 141, 0.96)'
          : 'rgba(133, 226, 203, 0.9)'
        context.textAlign = 'left'
        context.fillText(String(reaction.count), x + 66, REACTION_ROW_TOP + 39)
      }
    })
  } else {
    context.fillStyle = 'rgba(75, 201, 174, 0.1)'
    context.fillRect(34, REACTION_ROW_TOP, 956, 72)
    context.strokeStyle = 'rgba(98, 218, 186, 0.35)'
    context.strokeRect(34.5, REACTION_ROW_TOP + 0.5, 955, 71)
    context.font = '600 27px "Noto Sans SC", "PingFang SC", sans-serif'
    context.fillStyle = 'rgba(116, 224, 198, 0.88)'
    context.textAlign = 'center'
    context.textBaseline = 'middle'
    context.fillText('点击展开完整描述变更', 512, REACTION_ROW_TOP + 37)
  }
  context.textAlign = 'left'
  context.textBaseline = 'middle'

  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  return texture
}

function progressTextureKey(item: ProgressItem): string {
  if (item.kind === 'description_change') {
    const change = item.descriptionChange
    return [
      progressKey(item),
      change.author.userName,
      change.createdAt,
      change.before,
      change.after,
    ].join('|')
  }
  const comment = item.comment
  const reactions = comment.reactions
    .map((reaction) =>
      `${reaction.emoji}:${reaction.count}:${reaction.reactedByMe ? 1 : 0}`,
    )
    .join(',')
  return [
    progressKey(item),
    comment.author.userName,
    comment.createdAt,
    comment.body,
    comment.attachments.length,
    reactions,
  ].join('|')
}

function progressKey(item: ProgressItem): string {
  return item.kind === 'comment'
    ? `comment:${item.comment.id}`
    : `description_change:${item.descriptionChange.id}`
}

function createEmptyTexture(theme: ThemeMode): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = 1024
  canvas.height = 160
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context is unavailable')
  context.clearRect(0, 0, canvas.width, canvas.height)
  context.font = '600 52px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = CARD_PALETTES[theme].empty
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText('该节点还没有发布进展', canvas.width / 2, canvas.height / 2)
  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  return texture
}

function wrapTextPreserving(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  maxLines: number,
): { lines: string[]; truncated: boolean } {
  const lines: string[] = []
  const logicalLines = text
    .replace(/\r\n?/g, '\n')
    .replace(/\t/g, '    ')
    .split('\n')
  let truncated = false

  outer:
  for (let lineIndex = 0; lineIndex < logicalLines.length; lineIndex++) {
    const logicalLine = logicalLines[lineIndex] ?? ''
    if (!logicalLine) {
      lines.push('')
      if (lines.length >= maxLines) {
        truncated = lineIndex < logicalLines.length - 1
        break
      }
      continue
    }

    let current = ''
    for (const character of Array.from(logicalLine)) {
      const candidate = current + character
      if (!current || context.measureText(candidate).width <= maxWidth) {
        current = candidate
        continue
      }
      lines.push(current)
      current = character
      if (lines.length >= maxLines) {
        truncated = true
        break outer
      }
    }
    if (current) lines.push(current)
    if (lines.length >= maxLines) {
      truncated = lineIndex < logicalLines.length - 1
      break
    }
  }

  if (lines.length > maxLines) lines.length = maxLines
  if (truncated && lines.length) {
    let last = lines[maxLines - 1] ?? ''
    while (last && context.measureText(`${last}…`).width > maxWidth) {
      last = Array.from(last).slice(0, -1).join('')
    }
    lines[maxLines - 1] = `${last}…`
  }
  return { lines, truncated }
}

function trimToWidth(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  fontSize: number,
): string {
  context.font = `600 ${fontSize}px "Noto Sans SC", "PingFang SC", sans-serif`
  if (context.measureText(text).width <= maxWidth) return text
  let value = text
  while (value && context.measureText(`${value}…`).width > maxWidth) {
    value = Array.from(value).slice(0, -1).join('')
  }
  return `${value}…`
}

function smoothstep(edge0: number, edge1: number, value: number): number {
  const ratio = Math.min(1, Math.max(0, (value - edge0) / (edge1 - edge0)))
  return ratio * ratio * (3 - 2 * ratio)
}
