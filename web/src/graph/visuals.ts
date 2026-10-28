import * as THREE from 'three'

const LABEL_CANVAS_WIDTH = 768
const LABEL_CANVAS_HEIGHT = 192

export const LABEL_ASPECT_RATIO = LABEL_CANVAS_WIDTH / LABEL_CANVAS_HEIGHT

export function createLabelTexture(
  title: string,
  completed: boolean,
  virtual: boolean,
): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = LABEL_CANVAS_WIDTH
  canvas.height = LABEL_CANVAS_HEIGHT
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context is unavailable')

  context.clearRect(0, 0, canvas.width, canvas.height)
  roundedRect(context, 8, 8, 752, 176, 16)
  context.fillStyle = virtual
    ? 'rgba(17, 15, 10, 0.9)'
    : 'rgba(7, 13, 16, 0.84)'
  context.fill()
  context.strokeStyle = virtual
    ? 'rgba(246, 191, 74, 0.82)'
    : completed
      ? 'rgba(92, 223, 182, 0.5)'
      : 'rgba(239, 181, 67, 0.42)'
  context.lineWidth = 2
  context.stroke()

  context.fillStyle = virtual ? '#f4bd4e' : completed ? '#68dfb7' : '#e8b34a'
  context.fillRect(26, 30, 4, 132)

  context.textBaseline = 'middle'
  context.textAlign = 'left'
  context.font = '800 42px Inter, "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = virtual ? '#f4bd4e' : 'rgba(178, 199, 206, 0.82)'
  context.fillText(
    virtual ? 'GUILD CORE' : completed ? 'SYNCED' : 'ACTIVE',
    54,
    42,
  )

  context.font = '700 64px Inter, "Noto Sans SC", "PingFang SC", sans-serif'
  context.fillStyle = '#f4f1e8'
  const lines = wrapText(context, title || '未命名节点', 664, 1)
  context.fillText(lines[0] ?? '未命名节点', 54, 110)

  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  return texture
}

export function createEdgeGeometry(
  from: THREE.Vector3,
  to: THREE.Vector3,
): THREE.BufferGeometry {
  const middle = from.clone().lerp(to, 0.5)
  middle.y += 0.12 + from.distanceTo(to) * 0.008
  const curve = new THREE.QuadraticBezierCurve3(
    from.clone().setY(0.16),
    middle,
    to.clone().setY(0.16),
  )
  return new THREE.BufferGeometry().setFromPoints(curve.getPoints(30))
}

export function createNodeGlowTexture(): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = 256
  canvas.height = 256
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context is unavailable')
  const gradient = context.createRadialGradient(128, 128, 4, 128, 128, 126)
  gradient.addColorStop(0, 'rgba(255, 242, 190, 0.75)')
  gradient.addColorStop(0.22, 'rgba(241, 181, 63, 0.42)')
  gradient.addColorStop(0.62, 'rgba(211, 145, 33, 0.1)')
  gradient.addColorStop(1, 'rgba(211, 145, 33, 0)')
  context.fillStyle = gradient
  context.fillRect(0, 0, 256, 256)
  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  return texture
}

export function createDust(): THREE.Points<THREE.BufferGeometry, THREE.PointsMaterial> {
  const positions = new Float32Array(760 * 3)
  for (let index = 0; index < 760; index++) {
    const radius = 28 + Math.random() * 230
    const angle = Math.random() * Math.PI * 2
    positions[index * 3] = Math.cos(angle) * radius
    positions[index * 3 + 1] = Math.random() * 66 + 1
    positions[index * 3 + 2] = Math.sin(angle) * radius
  }
  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
  const material = new THREE.PointsMaterial({
    color: 0xd8ad60,
    size: 0.18,
    transparent: true,
    opacity: 0.28,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
  })
  return new THREE.Points(geometry, material)
}

function roundedRect(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
) {
  context.beginPath()
  context.moveTo(x + radius, y)
  context.lineTo(x + width - radius, y)
  context.quadraticCurveTo(x + width, y, x + width, y + radius)
  context.lineTo(x + width, y + height - radius)
  context.quadraticCurveTo(x + width, y + height, x + width - radius, y + height)
  context.lineTo(x + radius, y + height)
  context.quadraticCurveTo(x, y + height, x, y + height - radius)
  context.lineTo(x, y + radius)
  context.quadraticCurveTo(x, y, x + radius, y)
  context.closePath()
}

function wrapText(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  maxLines: number,
): string[] {
  const characters = Array.from(text.trim())
  const lines: string[] = []
  let current = ''
  for (const character of characters) {
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
  if (lines.length === maxLines && characters.join('') !== lines.join('')) {
    let last = lines[maxLines - 1] ?? ''
    while (last && context.measureText(`${last}…`).width > maxWidth) {
      last = Array.from(last).slice(0, -1).join('')
    }
    lines[maxLines - 1] = `${last}…`
  }
  return lines.length > 0 ? lines : ['未命名节点']
}
