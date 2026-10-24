import * as THREE from 'three'
import type { Comment, GraphNeighborhood, GraphNode } from '@/lib/api'
import {
  layoutGraph,
  VIRTUAL_ROOT_ID,
  type GraphLayout,
} from './layout'
import { CommentLayer } from './CommentLayer'
import { createDust, createEdgeGeometry, createLabelTexture } from './visuals'

interface SceneCallbacks {
  onSelect: (id: number) => void
  onBlank: () => void
  onCommentsNeedOlder: () => void
}

interface NodeVisual {
  group: THREE.Group
  base: THREE.Mesh<THREE.SphereGeometry, THREE.MeshStandardMaterial>
  label: THREE.Sprite
  radius: number
  targetPosition: THREE.Vector3
  targetOpacity: number
  opacity: number
  labelScale: number
  labelEmphasis: number
}

interface EdgeVisual {
  line: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>
  parentId: number
  childId: number
  targetOpacity: number
}

export class GraphScene {
  private readonly container: HTMLElement
  private readonly callbacks: SceneCallbacks
  private readonly scene = new THREE.Scene()
  private readonly camera = new THREE.PerspectiveCamera(44, 1, 0.1, 1800)
  private readonly renderer: THREE.WebGLRenderer
  private readonly world = new THREE.Group()
  private readonly nodeViews = new Map<number, NodeVisual>()
  private readonly edgeViews = new Map<string, EdgeVisual>()
  private readonly clickTargets: THREE.Mesh[] = []
  private readonly dataNodes = new Map<number, GraphNode>()
  private readonly extraNodes = new Map<number, GraphNode>()
  private readonly sphereGeometry = new THREE.SphereGeometry(1, 28, 20)
  private readonly commentLayer = new CommentLayer()
  private readonly raycaster = new THREE.Raycaster()
  private readonly pointerNdc = new THREE.Vector2()
  private readonly viewTarget = new THREE.Vector3()
  private readonly desiredTarget = new THREE.Vector3()
  private readonly dust: THREE.Points
  private readonly resizeObserver: ResizeObserver
  private layout: GraphLayout = layoutGraph([])
  private focus: GraphNeighborhood | null = null
  private focusedIds = new Set<number>()
  private selectedId: number | null = null
  private snapshot: {
    target: THREE.Vector3
    distance: number
    pitch: number
    yaw: number
  } | null = null
  private distance = 90
  private desiredDistance = 90
  private pitch = THREE.MathUtils.degToRad(78)
  private desiredPitch = this.pitch
  private yaw = 0
  private desiredYaw = 0
  private focusNodePosition: THREE.Vector3 | null = null
  private pointerX = 0
  private pointerY = 0
  private smoothPointerX = 0
  private smoothPointerY = 0
  private dragging = false
  private dragDistance = 0
  private lastPointer = new THREE.Vector2()
  private hasInteracted = false
  private frameId = 0
  private lastFrame = performance.now()
  private disposed = false

  constructor(container: HTMLElement, callbacks: SceneCallbacks) {
    this.container = container
    this.callbacks = callbacks
    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: false,
      powerPreference: 'high-performance',
    })
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.toneMapping = THREE.NoToneMapping
    this.renderer.domElement.className = 'block h-full w-full cursor-grab'
    this.renderer.domElement.style.touchAction = 'none'
    container.appendChild(this.renderer.domElement)

    this.scene.background = new THREE.Color(0xffffff)
    this.scene.fog = new THREE.FogExp2(0xffffff, 0.003)
    this.world.add(this.createGround())
    this.scene.add(this.world)
    this.scene.add(this.commentLayer.group)
    this.dust = createDust()
    this.scene.add(this.dust)
    this.scene.add(new THREE.HemisphereLight(0xffffff, 0xc8d5e1, 2))
    const key = new THREE.DirectionalLight(0xffffff, 2)
    key.position.set(24, 50, 30)
    this.scene.add(key)

    this.camera.position.set(0, 88, 20)
    this.camera.lookAt(0, 0, 0)
    this.resizeObserver = new ResizeObserver(() => this.resize())
    this.resizeObserver.observe(container)
    this.bindEvents()
    this.resize()
    this.animate()
  }

  setNodes(nodes: GraphNode[]) {
    this.dataNodes.clear()
    for (const node of nodes) this.dataNodes.set(node.id, node)
    this.rebuildViews()
    if (!this.hasInteracted && !this.focus) this.fitView()
  }

  setComments(comments: Comment[], loading = false) {
    this.commentLayer.setComments(comments, loading)
  }

  scrollComments(delta: number) {
    if (!this.focus) return
    if (this.commentLayer.scrollBy(delta)) {
      this.callbacks.onCommentsNeedOlder()
    }
  }

  setFocus(neighborhood: GraphNeighborhood | null) {
    if (!neighborhood) {
      this.clearFocus()
      return
    }
    const related = [
      neighborhood.node,
      ...(neighborhood.parent ? [neighborhood.parent] : []),
      ...neighborhood.children,
    ]
    for (const node of related) this.extraNodes.set(node.id, node)

    const ids = new Set(related.map((node) => node.id))
    if (!neighborhood.parent) ids.add(VIRTUAL_ROOT_ID)
    if (!this.focus && !this.snapshot) {
      this.snapshot = {
        target: this.viewTarget.clone(),
        distance: this.distance,
        pitch: this.pitch,
        yaw: this.yaw,
      }
    }
    this.focus = neighborhood
    this.selectedId = neighborhood.node.id
    this.focusedIds = ids
    this.desiredDistance = 18
    this.desiredPitch = THREE.MathUtils.degToRad(15)
    this.desiredYaw = (this.snapshot?.yaw ?? this.yaw) + THREE.MathUtils.degToRad(30)
    this.focusNodePosition = new THREE.Vector3(
      this.layout.positions.get(this.selectedId)?.x ?? 0,
      0.9,
      this.layout.positions.get(this.selectedId)?.z ?? 0,
    )
    this.desiredTarget.copy(this.focusTarget(
      this.focusNodePosition,
      this.desiredYaw,
      this.desiredPitch,
      this.desiredDistance,
    ))
    this.hasInteracted = true
    this.rebuildViews()
  }

  clearFocus() {
    if (!this.focus) return
    this.focus = null
    this.selectedId = null
    this.focusNodePosition = null
    this.focusedIds.clear()
    this.hasInteracted = true
    if (this.snapshot) {
      this.desiredTarget.copy(this.snapshot.target)
      this.desiredDistance = this.snapshot.distance
      this.desiredPitch = this.snapshot.pitch
      this.desiredYaw = this.snapshot.yaw
      this.snapshot = null
    }
    this.applyVisualState()
  }

  fitView() {
    const verticalFov = THREE.MathUtils.degToRad(this.camera.fov)
    const horizontalFov = 2 * Math.atan(Math.tan(verticalFov / 2) * this.camera.aspect)
    const horizontalDistance = this.layout.width / (2 * Math.tan(horizontalFov / 2))
    const verticalDistance = this.layout.depth / (2 * Math.tan(verticalFov / 2))
    this.desiredTarget.set(this.layout.centerX, 0, this.layout.centerZ)
    this.viewTarget.copy(this.desiredTarget)
    this.desiredDistance = Math.max(24, horizontalDistance, verticalDistance) * 1.28
    this.distance = this.desiredDistance
    this.desiredPitch = THREE.MathUtils.degToRad(78)
    this.pitch = this.desiredPitch
    this.desiredYaw = 0
    this.yaw = 0
    this.focusNodePosition = null
  }

  destroy() {
    this.disposed = true
    cancelAnimationFrame(this.frameId)
    this.resizeObserver.disconnect()
    this.unbindEvents()
    for (const id of [...this.nodeViews.keys()]) this.removeNode(id)
    for (const edge of this.edgeViews.values()) this.disposeEdge(edge)
    this.edgeViews.clear()
    this.commentLayer.dispose()
    this.world.traverse((object) => {
      if (object === this.world) return
      if (object instanceof THREE.Mesh || object instanceof THREE.GridHelper) {
        object.geometry.dispose()
        const material: THREE.Material | THREE.Material[] = object.material
        const materials = Array.isArray(material) ? material : [material]
        materials.forEach((material) => material.dispose())
      }
    })
    this.sphereGeometry.dispose()
    this.dust.geometry.dispose()
    if (Array.isArray(this.dust.material)) {
      this.dust.material.forEach((material) => material.dispose())
    } else {
      this.dust.material.dispose()
    }
    this.renderer.dispose()
    this.renderer.domElement.remove()
  }

  private rebuildViews() {
    const nodes = [...this.extraNodes.values()]
    const known = new Set(nodes.map((node) => node.id))
    for (const node of this.dataNodes.values()) {
      if (!known.has(node.id)) nodes.push(node)
    }
    nodes.push({
      id: VIRTUAL_ROOT_ID,
      parentId: null,
      parentTitle: null,
      title: '公会',
      sortOrder: -1,
      completedAt: null,
      childCount: nodes.filter((node) => node.parentId == null).length,
      commentCount: 0,
    })
    this.layout = layoutGraph(nodes)

    const activeIds = new Set(nodes.map((node) => node.id))
    for (const id of [...this.nodeViews.keys()]) {
      if (!activeIds.has(id)) this.removeNode(id)
    }
    for (const node of nodes) {
      const position = this.layout.positions.get(node.id)
      if (!position) continue
      let view = this.nodeViews.get(node.id)
      if (!view) {
        view = this.createNode(node)
        this.nodeViews.set(node.id, view)
        this.world.add(view.group)
      }
      view.targetPosition.set(position.x, 0.9, position.z)
    }
    this.rebuildEdges()
    this.applyVisualState()
  }

  private createNode(node: GraphNode): NodeVisual {
    const virtual = node.id === VIRTUAL_ROOT_ID
    const radius = virtual ? 0.88 : 0.64
    const material = new THREE.MeshStandardMaterial({
      color: virtual ? 0x5d89ad : node.completedAt ? 0x4d9875 : 0x5687b5,
      emissive: virtual ? 0x1a4c70 : node.completedAt ? 0x13553b : 0x1b537e,
      emissiveIntensity: virtual ? 0.34 : 0.25,
      roughness: 0.34,
      metalness: 0.18,
      transparent: true,
      opacity: 0,
    })
    const group = new THREE.Group()
    const base = new THREE.Mesh(this.sphereGeometry, material)
    base.scale.setScalar(radius)
    base.userData.nodeId = node.id
    const labelMaterial = new THREE.SpriteMaterial({
      map: createLabelTexture(node.title, node.completedAt != null, virtual),
      transparent: true,
      opacity: 0,
      depthWrite: false,
      toneMapped: false,
    })
    const label = new THREE.Sprite(labelMaterial)
    label.position.y = 1.17
    label.scale.set(4.45, 1.11, 1)
    group.add(base, label)
    if (!virtual) this.clickTargets.push(base)
    const position = this.layout.positions.get(node.id)
    group.position.set(position?.x ?? 0, 0.9, position?.z ?? 0)
    return {
      group,
      base,
      label,
      radius,
      targetPosition: group.position.clone(),
      targetOpacity: 0,
      opacity: 0,
      labelScale: 1,
      labelEmphasis: 1,
    }
  }

  private removeNode(id: number) {
    const view = this.nodeViews.get(id)
    if (!view) return
    this.world.remove(view.group)
    view.label.material.map?.dispose()
    view.label.material.dispose()
    view.base.material.dispose()
    const index = this.clickTargets.indexOf(view.base)
    if (index >= 0) this.clickTargets.splice(index, 1)
    this.nodeViews.delete(id)
  }

  private rebuildEdges() {
    for (const [id, edge] of this.edgeViews) {
      this.world.remove(edge.line)
      this.disposeEdge(edge)
      this.edgeViews.delete(id)
    }
    for (const edge of this.layout.edges) {
      const parent = this.layout.positions.get(edge.parentId)
      const child = this.layout.positions.get(edge.childId)
      if (!parent || !child) continue
      const material = new THREE.LineBasicMaterial({
        color: 0x6d899f,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        blending: THREE.NormalBlending,
      })
      const geometry = createEdgeGeometry(
        new THREE.Vector3(parent.x, 0, parent.z),
        new THREE.Vector3(child.x, 0, child.z),
      )
      const line = new THREE.Line(geometry, material)
      this.world.add(line)
      this.edgeViews.set(edge.id, {
        line,
        parentId: edge.parentId,
        childId: edge.childId,
        targetOpacity: 0.32,
      })
    }
  }

  private disposeEdge(edge: EdgeVisual) {
    edge.line.geometry.dispose()
    edge.line.material.dispose()
  }

  private applyVisualState() {
    for (const [id, view] of this.nodeViews) {
      const visible = !this.focus || this.focusedIds.has(id)
      view.targetOpacity = visible ? (id === this.selectedId ? 1 : id === VIRTUAL_ROOT_ID ? 0.72 : 0.94) : 0
      if (id === this.selectedId) {
        view.base.material.emissive.setHex(0x2b78a8)
        view.base.material.emissiveIntensity = 0.5
      } else {
        view.base.material.emissive.setHex(0x275b7e)
        view.base.material.emissiveIntensity = 0.2
      }
    }
    for (const edge of this.edgeViews.values()) {
      edge.targetOpacity = !this.focus
        ? 0.32
        : this.focusedIds.has(edge.parentId) && this.focusedIds.has(edge.childId)
          ? 0.82
          : 0
    }
  }

  private createGround(): THREE.Group {
    const ground = new THREE.Group()
    const plane = new THREE.Mesh(
      new THREE.PlaneGeometry(1000, 1000),
      new THREE.MeshStandardMaterial({
        color: 0xffffff,
        roughness: 1,
        metalness: 0,
      }),
    )
    plane.rotation.x = -Math.PI / 2
    plane.position.y = -0.12
    const grid = new THREE.Mesh(
      new THREE.PlaneGeometry(1000, 1000),
      new THREE.ShaderMaterial({
        transparent: true,
        depthWrite: false,
        side: THREE.DoubleSide,
        uniforms: {
          gridColor: { value: new THREE.Color(0x91a5b8) },
          gridSpacing: { value: 4 },
          majorSpacing: { value: 20 },
          gridOpacity: { value: 0.34 },
          fadeStart: { value: 38 },
          fadeEnd: { value: 145 },
        },
        vertexShader: `
          varying vec3 vWorldPosition;
          void main() {
            vec4 worldPosition = modelMatrix * vec4(position, 1.0);
            vWorldPosition = worldPosition.xyz;
            gl_Position = projectionMatrix * viewMatrix * worldPosition;
          }
        `,
        fragmentShader: `
          uniform vec3 gridColor;
          uniform float gridSpacing;
          uniform float majorSpacing;
          uniform float gridOpacity;
          uniform float fadeStart;
          uniform float fadeEnd;
          varying vec3 vWorldPosition;

          float gridLine(vec2 worldPosition, float spacing) {
            vec2 coordinate = worldPosition / spacing;
            vec2 derivative = fwidth(coordinate);
            vec2 distanceToLine = abs(fract(coordinate - 0.5) - 0.5) / max(derivative, vec2(0.0001));
            return 1.0 - min(min(distanceToLine.x, distanceToLine.y), 1.0);
          }

          void main() {
            float minorLine = gridLine(vWorldPosition.xz, gridSpacing);
            float majorLine = gridLine(vWorldPosition.xz, majorSpacing);
            float distanceFromCamera = distance(vWorldPosition.xz, cameraPosition.xz);
            float distanceFade = 1.0 - smoothstep(fadeStart, fadeEnd, distanceFromCamera);
            float alpha = max(minorLine * 0.55, majorLine) * distanceFade * gridOpacity;
            if (alpha < 0.002) discard;
            gl_FragColor = vec4(gridColor, alpha);
          }
        `,
      }),
    )
    grid.rotation.x = -Math.PI / 2
    grid.position.y = -0.1
    ground.add(plane, grid)
    return ground
  }

  private bindEvents() {
    this.renderer.domElement.addEventListener('pointerdown', this.onPointerDown)
    this.renderer.domElement.addEventListener('pointermove', this.onPointerMove)
    this.renderer.domElement.addEventListener('pointerup', this.onPointerUp)
    this.renderer.domElement.addEventListener('pointercancel', this.onPointerUp)
    this.renderer.domElement.addEventListener('pointerleave', this.onPointerLeave)
    this.renderer.domElement.addEventListener('wheel', this.onWheel, { passive: false })
  }

  private unbindEvents() {
    this.renderer.domElement.removeEventListener('pointerdown', this.onPointerDown)
    this.renderer.domElement.removeEventListener('pointermove', this.onPointerMove)
    this.renderer.domElement.removeEventListener('pointerup', this.onPointerUp)
    this.renderer.domElement.removeEventListener('pointercancel', this.onPointerUp)
    this.renderer.domElement.removeEventListener('pointerleave', this.onPointerLeave)
    this.renderer.domElement.removeEventListener('wheel', this.onWheel)
  }

  private onPointerDown = (event: PointerEvent) => {
    if (event.button !== 0) return
    this.dragging = true
    this.dragDistance = 0
    this.lastPointer.set(event.clientX, event.clientY)
    this.renderer.domElement.setPointerCapture(event.pointerId)
    this.renderer.domElement.classList.add('cursor-grabbing')
  }

  private onPointerMove = (event: PointerEvent) => {
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.pointerX = ((event.clientX - rect.left) / rect.width) * 2 - 1
    this.pointerY = ((event.clientY - rect.top) / rect.height) * 2 - 1
    if (!this.dragging) return
    const dx = event.clientX - this.lastPointer.x
    const dy = event.clientY - this.lastPointer.y
    this.dragDistance += Math.abs(dx) + Math.abs(dy)
    this.lastPointer.set(event.clientX, event.clientY)
    if (this.focus || this.dragDistance < 3) return
    this.hasInteracted = true
    const scale = this.distance / Math.max(rect.height, 1) * 1.15
    this.desiredTarget.x -= dx * scale
    this.desiredTarget.z -= dy * scale
    this.renderer.domElement.classList.add('cursor-grabbing')
  }

  private onPointerUp = (event: PointerEvent) => {
    if (!this.dragging) return
    this.dragging = false
    this.renderer.domElement.classList.remove('cursor-grabbing')
    if (this.renderer.domElement.hasPointerCapture(event.pointerId)) {
      this.renderer.domElement.releasePointerCapture(event.pointerId)
    }
    if (this.dragDistance > 5) return
    const id = this.pickNode(event)
    if (id == null) this.callbacks.onBlank()
    else this.callbacks.onSelect(id)
  }

  private onPointerLeave = () => {
    if (!this.dragging) {
      this.pointerX = 0
      this.pointerY = 0
    }
  }

  private onWheel = (event: WheelEvent) => {
    if (this.focus) return
    event.preventDefault()
    this.hasInteracted = true
    this.desiredDistance = THREE.MathUtils.clamp(
      this.desiredDistance * Math.exp(event.deltaY * 0.0011),
      16,
      440,
    )
  }

  private pickNode(event: PointerEvent): number | null {
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.pointerNdc.set(
      ((event.clientX - rect.left) / rect.width) * 2 - 1,
      -((event.clientY - rect.top) / rect.height) * 2 + 1,
    )
    this.raycaster.setFromCamera(this.pointerNdc, this.camera)
    for (const hit of this.raycaster.intersectObjects(this.clickTargets, false)) {
      const id = hit.object.userData.nodeId
      const view = typeof id === 'number' ? this.nodeViews.get(id) : undefined
      if (typeof id === 'number' && view && view.opacity > 0.25) return id
    }
    return null
  }

  private focusTarget(
    nodePosition: THREE.Vector3,
    yaw: number,
    pitch: number,
    distance: number,
  ): THREE.Vector3 {
    const verticalExtent = 2 * distance * Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2))
    const horizontalExtent = verticalExtent * this.camera.aspect
    const offsetDirection = new THREE.Vector3(
      Math.sin(yaw) * Math.cos(pitch),
      Math.sin(pitch),
      Math.cos(yaw) * Math.cos(pitch),
    )
    const forward = offsetDirection.multiplyScalar(-1).normalize()
    const right = forward.clone().cross(new THREE.Vector3(0, 1, 0)).normalize()
    const screenUp = right.clone().cross(forward).normalize()
    return nodePosition.clone()
      .addScaledVector(right, horizontalExtent * 0.11)
      .addScaledVector(screenUp, verticalExtent * 0.24)
  }

  private resize() {
    const width = Math.max(1, this.container.clientWidth)
    const height = Math.max(1, this.container.clientHeight)
    this.camera.aspect = width / height
    this.camera.updateProjectionMatrix()
    this.renderer.setSize(width, height, false)
    if (this.focus && this.focusNodePosition) {
      this.desiredTarget.copy(this.focusTarget(
        this.focusNodePosition,
        this.desiredYaw,
        this.desiredPitch,
        this.desiredDistance,
      ))
    }
  }

  private animate = () => {
    if (this.disposed) return
    this.frameId = requestAnimationFrame(this.animate)
    const now = performance.now()
    const delta = Math.min(0.05, (now - this.lastFrame) / 1000)
    this.lastFrame = now
    const damping = 1 - Math.exp(-delta * 8)
    const opacityDamping = 1 - Math.exp(-delta * 7)

    this.smoothPointerX += (this.pointerX - this.smoothPointerX) * damping
    this.smoothPointerY += (this.pointerY - this.smoothPointerY) * damping
    this.world.rotation.z = -this.smoothPointerX * 0.018
    this.world.rotation.x = this.smoothPointerY * 0.014
    this.dust.rotation.y += delta * 0.004

    this.viewTarget.lerp(this.desiredTarget, damping)
    this.distance += (this.desiredDistance - this.distance) * damping
    this.pitch += (this.desiredPitch - this.pitch) * damping
    this.yaw += (this.desiredYaw - this.yaw) * damping
    const horizontalDistance = Math.cos(this.pitch) * this.distance
    const cameraOffset = new THREE.Vector3(
      Math.sin(this.yaw) * horizontalDistance,
      Math.sin(this.pitch) * this.distance,
      Math.cos(this.yaw) * horizontalDistance,
    )
    this.camera.position.copy(this.viewTarget).add(cameraOffset)
    this.camera.lookAt(this.viewTarget)

    for (const [id, view] of this.nodeViews) {
      view.group.position.lerp(view.targetPosition, damping)
      view.opacity += (view.targetOpacity - view.opacity) * opacityDamping
      const neighborTitle = Boolean(
        this.focus && id !== this.selectedId && this.focusedIds.has(id),
      )
      view.labelScale += ((neighborTitle ? 0.82 : 1) - view.labelScale) * damping
      view.labelEmphasis += ((neighborTitle ? 0.6 : 1) - view.labelEmphasis) * opacityDamping
      view.base.material.opacity = view.opacity
      view.label.scale.set(4.45 * view.labelScale, 1.11 * view.labelScale, 1)
      view.label.material.opacity = Math.min(1, view.opacity * 1.12) * view.labelEmphasis
      const pulse = id === this.selectedId ? 1.08 + Math.sin(now * 0.004) * 0.035 : 1
      view.base.scale.setScalar(view.radius * pulse)
      const visible = view.opacity > 0.01
      view.group.visible = visible
      if (visible) {
        view.base.material.opacity = view.opacity
        view.label.material.opacity = Math.min(1, view.opacity * 1.12) * view.labelEmphasis
      }
    }
    for (const edge of this.edgeViews.values()) {
      const material = edge.line.material
      material.opacity += (edge.targetOpacity - material.opacity) * opacityDamping
      edge.line.visible = material.opacity > 0.01
    }

    const selected = this.focus ? this.nodeViews.get(this.focus.node.id) : null
    if (selected?.group.visible) {
      selected.group.updateWorldMatrix(true, false)
      const nodeWorldPosition = new THREE.Vector3()
      selected.group.getWorldPosition(nodeWorldPosition)
      this.commentLayer.update(delta, this.camera, nodeWorldPosition, true)
    } else {
      this.commentLayer.update(delta, this.camera, new THREE.Vector3(), false)
    }
    this.renderer.render(this.scene, this.camera)
  }
}
