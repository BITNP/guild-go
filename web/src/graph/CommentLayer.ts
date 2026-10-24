import * as THREE from 'three'
import type { Comment } from '@/lib/api'

interface CommentVisual {
  anchor: THREE.Group
  card: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>
  dot: THREE.Mesh<THREE.SphereGeometry, THREE.MeshBasicMaterial>
  line: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>
  y: number
  targetY: number
  opacity: number
}

const CARD_WIDTH = 6.2
const CARD_HEIGHT = 1.95
const CARD_CENTER_X = CARD_WIDTH / 2 + 0.28
const BASE_Y = 2.45
const SPACING = 2.25
const FADE_START = 6.2
const FADE_END = 15.5

export class CommentLayer {
  readonly group = new THREE.Group()
  private readonly cardGeometry = new THREE.PlaneGeometry(CARD_WIDTH, CARD_HEIGHT)
  private readonly dotGeometry = new THREE.SphereGeometry(0.15, 16, 12)
  private readonly emptyLabel: THREE.Sprite
  private readonly visuals = new Map<number, CommentVisual>()
  private targetScroll = 0
  private scroll = 0
  private lastNodePosition: THREE.Vector3 | null = null
  private commentsLoading = false
  private emptyOpacity = 0

  constructor() {
    this.group.visible = false
    this.group.renderOrder = 5
    this.emptyLabel = new THREE.Sprite(new THREE.SpriteMaterial({
      map: createEmptyTexture(),
      transparent: true,
      opacity: 0,
      depthWrite: false,
      toneMapped: false,
    }))
    this.emptyLabel.scale.set(6.2, 0.98, 1)
    this.emptyLabel.visible = false
    this.group.add(this.emptyLabel)
  }

  setComments(comments: Comment[], loading = false) {
    this.commentsLoading = loading
    const previousCount = this.visuals.size
    const active = new Set(comments.map((comment) => comment.id))
    for (const [id, visual] of this.visuals) {
      if (active.has(id)) continue
      this.group.remove(visual.anchor, visual.line)
      this.disposeVisual(visual)
      this.visuals.delete(id)
    }

    comments.forEach((comment, index) => {
      let visual = this.visuals.get(comment.id)
      if (!visual) {
        visual = this.createVisual(comment)
        this.visuals.set(comment.id, visual)
        this.group.add(visual.anchor, visual.line)
      }
      visual.targetY = BASE_Y + (comments.length - 1 - index) * SPACING
    })

    if (previousCount > 0 && comments.length > previousCount) {
      const added = (comments.length - previousCount) * SPACING
      this.scroll -= added
      this.targetScroll -= added
    }
    this.clampScroll()
  }

  scrollBy(delta: number): boolean {
    this.targetScroll += delta * 0.022
    this.clampScroll()
    return this.targetScroll <= this.minScroll() + 0.15
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
      visual.card.material.opacity = visual.opacity * 0.88
      visual.dot.material.opacity = visual.opacity
      visual.line.material.opacity = visual.opacity * 0.3
      visual.card.visible = visual.opacity > 0.012
      visual.dot.visible = visual.opacity > 0.012
      visual.line.visible = visual.opacity > 0.012
    }

    const showEmpty = selected && !this.commentsLoading && this.visuals.size === 0
    this.emptyOpacity += ((showEmpty ? 1 : 0) - this.emptyOpacity) * opacityDamping
    this.emptyLabel.position.set(
      this.lastNodePosition.x,
      this.lastNodePosition.y + BASE_Y + 0.2,
      this.lastNodePosition.z,
    )
    this.emptyLabel.material.opacity = this.emptyOpacity
    this.emptyLabel.visible = this.emptyOpacity > 0.012
    this.group.visible = this.emptyOpacity > 0.012
      || [...this.visuals.values()].some((visual) => visual.opacity > 0.012)
  }

  dispose() {
    for (const visual of this.visuals.values()) this.disposeVisual(visual)
    this.visuals.clear()
    this.cardGeometry.dispose()
    this.dotGeometry.dispose()
    this.emptyLabel.material.map?.dispose()
    this.emptyLabel.material.dispose()
    this.group.clear()
  }

  private createVisual(comment: Comment): CommentVisual {
    const texture = createCommentTexture(comment)
    const card = new THREE.Mesh(
      this.cardGeometry,
      new THREE.MeshBasicMaterial({
        map: texture,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        side: THREE.DoubleSide,
      }),
    )
    const dot = new THREE.Mesh(
      this.dotGeometry,
      new THREE.MeshBasicMaterial({
        color: comment.reactions.length > 0 ? 0x2b86ba : 0x4c789c,
        transparent: true,
        opacity: 0,
        depthWrite: false,
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
        color: 0x6f91ad,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        blending: THREE.NormalBlending,
      }),
    )
    return {
      anchor,
      card,
      dot,
      line,
      y: BASE_Y,
      targetY: BASE_Y,
      opacity: 0,
    }
  }

  private disposeVisual(visual: CommentVisual) {
    visual.card.material.map?.dispose()
    visual.card.material.dispose()
    visual.dot.material.dispose()
    visual.line.geometry.dispose()
    visual.line.material.dispose()
  }

  private minScroll(): number {
    return -Math.max(0, (this.visuals.size - 1) * SPACING)
  }

  private clampScroll() {
    const minimum = this.minScroll()
    this.targetScroll = Math.max(minimum, Math.min(0, this.targetScroll))
    this.scroll = Math.max(minimum, Math.min(0, this.scroll))
  }
}

function createCommentTexture(comment: Comment): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = 1024
  canvas.height = 320
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context is unavailable')

  context.clearRect(0, 0, canvas.width, canvas.height)
  context.fillStyle = 'rgba(255, 255, 255, 0.74)'
  context.fillRect(0, 0, canvas.width, canvas.height)
  context.fillStyle = 'rgba(45, 114, 168, 0.78)'
  context.fillRect(0, 0, 5, canvas.height)

  context.font = '600 46px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = '#1f4f73'
  context.textBaseline = 'middle'
  context.fillText(trimToWidth(context, comment.author.userName, 590, 46), 34, 42)
  context.font = '34px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = 'rgba(83, 108, 129, 0.65)'
  const time = new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(comment.createdAt))
  context.textAlign = 'right'
  context.fillText(time, 990, 42)
  context.textAlign = 'left'

  context.font = '44px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = 'rgba(31, 53, 74, 0.92)'
  const body = comment.body.trim() || (comment.attachments.length > 0 ? '图片评注' : '无正文')
  const lines = wrapText(context, body, 940, 4)
  lines.forEach((line, index) => context.fillText(line, 34, 96 + index * 52))
  if (comment.attachments.length > 0) {
    context.font = '30px "Noto Sans SC", "PingFang SC", sans-serif'
    context.fillStyle = 'rgba(52, 114, 159, 0.72)'
    context.textAlign = 'right'
    context.fillText(`附 ${comment.attachments.length} 张图片`, 990, 292)
    context.textAlign = 'left'
  }
  context.strokeStyle = 'rgba(84, 124, 155, 0.24)'
  context.lineWidth = 1
  context.beginPath()
  context.moveTo(34, 76)
  context.lineTo(990, 76)
  context.stroke()

  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  return texture
}

function createEmptyTexture(): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = 1024
  canvas.height = 160
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context is unavailable')
  context.clearRect(0, 0, canvas.width, canvas.height)
  context.font = '600 52px "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = 'rgba(57, 86, 110, 0.82)'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText('该节点目前没有评论', canvas.width / 2, canvas.height / 2)
  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  return texture
}

function wrapText(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  maxLines: number,
): string[] {
  const lines: string[] = []
  let current = ''
  for (const character of Array.from(text.replace(/\s+/g, ' '))) {
    const candidate = current + character
    if (current && context.measureText(candidate).width > maxWidth) {
      lines.push(current)
      current = character
      if (lines.length === maxLines) break
    } else {
      current = candidate
    }
  }
  if (lines.length < maxLines && current) lines.push(current)
  if (lines.length === maxLines && Array.from(text).join('') !== lines.join('')) {
    let last = lines[maxLines - 1] ?? ''
    while (last && context.measureText(`${last}…`).width > maxWidth) {
      last = Array.from(last).slice(0, -1).join('')
    }
    lines[maxLines - 1] = `${last}…`
  }
  return lines
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
