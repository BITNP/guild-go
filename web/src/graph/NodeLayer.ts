import * as THREE from "three";

import type { ThemeMode } from "@/composables/useTheme";
import type { GraphNode } from "@/graph/types";

import type { GraphLayout } from "./layout";
import { VIRTUAL_ROOT_ID } from "./layout";
import { SCENE_PALETTES, nodeAccentColor } from "./palette";
import {
  LABEL_ASPECT_RATIO,
  createEdgeGeometry,
  createLabelTexture,
  createNodeGlowTexture,
} from "./visuals";

interface NodeVisual {
  group: THREE.Group;
  plate: THREE.Mesh<THREE.CylinderGeometry, THREE.MeshStandardMaterial>;
  cap: THREE.Mesh<THREE.CylinderGeometry, THREE.MeshStandardMaterial>;
  crystal: THREE.Mesh<THREE.OctahedronGeometry, THREE.MeshStandardMaterial>;
  innerRing: THREE.Mesh<THREE.RingGeometry, THREE.MeshBasicMaterial>;
  outerRing: THREE.Mesh<THREE.RingGeometry, THREE.MeshBasicMaterial>;
  glow: THREE.Sprite;
  particles: THREE.Points<THREE.BufferGeometry, THREE.PointsMaterial>;
  particleSeeds: Float32Array;
  label: THREE.Sprite | null;
  labelTargetPosition: THREE.Vector3;
  radius: number;
  accent: number;
  title: string;
  completed: boolean;
  baseEmissive: {
    plate: number;
    cap: number;
    crystal: number;
  };
  targetPosition: THREE.Vector3;
  targetOpacity: number;
  opacity: number;
  hoverAmount: number;
  labelScale: number;
  labelEmphasis: number;
  labelOpacity: number;
  targetLabelOpacity: number;
}

interface EdgeVisual {
  line: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>;
  parentId: number;
  childId: number;
  targetOpacity: number;
}

interface LabelCandidate {
  id: number;
  view: NodeVisual;
  depth: number;
  left: number;
  right: number;
  top: number;
  bottom: number;
  readable: boolean;
}

const LABEL_GAP = 10;
const LABEL_REAR_OFFSET = 2.15;
const LABEL_DEFAULT_OFFSET = 1.7;
const LABEL_COMMENT_SIDE_CLEARANCE = (4.8 * 1.1) / 2;
const HOVER_HIT_RADIUS_PX = 24;
const LABEL_ZOOM_HIDE_THRESHOLD = 0.68;
const HOVER_PARTICLE_COUNT = 18;
const HOVER_PARTICLE_STRIDE = 5;

export class NodeLayer {
  private readonly world: THREE.Group;
  private readonly nodeViews = new Map<number, NodeVisual>();
  private readonly edgeViews = new Map<string, EdgeVisual>();
  private readonly plateGeometry = new THREE.CylinderGeometry(1, 1, 0.16, 6);
  private readonly capGeometry = new THREE.CylinderGeometry(0.72, 0.72, 0.3, 6);
  private readonly crystalGeometry = new THREE.OctahedronGeometry(0.52, 0);
  private readonly innerRingGeometry = new THREE.RingGeometry(1.06, 1.15, 64);
  private readonly outerRingGeometry = new THREE.RingGeometry(
    1.36,
    1.41,
    64,
    1,
    0.22,
    Math.PI * 1.31,
  );

  private readonly glowTexture = createNodeGlowTexture();
  private readonly raycaster = new THREE.Raycaster();
  private readonly pickCenter = new THREE.Vector3();
  private readonly pickOffset = new THREE.Vector3();

  constructor(world: THREE.Group) {
    this.world = world;
  }

  public sync(nodes: GraphNode[], layout: GraphLayout, theme: ThemeMode) {
    const activeIds = new Set(nodes.map((node) => node.id));
    for (const id of this.nodeViews.keys()) {
      if (!activeIds.has(id)) {
        this.removeNode(id);
      }
    }
    for (const node of nodes) {
      const position = layout.positions.get(node.id);
      if (!position) {
        continue;
      }
      let view = this.nodeViews.get(node.id);
      if (!view) {
        view = this.createNode(node, layout, theme);
        this.nodeViews.set(node.id, view);
        this.world.add(view.group);
      } else if (
        view.completed !== (node.completedAt != null) ||
        view.title !== node.title
      ) {
        this.updateNodeAppearance(view, node, theme);
      }
      view.targetPosition.set(position.x, 0.9, position.z);
    }
    this.rebuildEdges(layout, theme);
  }

  public setTheme(theme: ThemeMode) {
    const palette = SCENE_PALETTES[theme];
    for (const edge of this.edgeViews.values()) {
      edge.line.material.color.setHex(palette.edge);
    }
    for (const [id, view] of this.nodeViews) {
      const virtual = id === VIRTUAL_ROOT_ID;
      const completed = view.completed && !virtual;
      view.accent = nodeAccentColor(theme, virtual, completed);
      view.plate.material.color.setHex(
        virtual ? palette.virtualPlate : palette.plate,
      );
      view.crystal.material.color.setHex(
        virtual ? palette.virtualCrystal : palette.crystal,
      );
      this.applyAccentColor(view, view.accent);
      if (view.label && !virtual) {
        this.refreshLabelTexture(view, view.title, completed, theme);
      }
    }
  }

  public applyVisualState(
    focusActive: boolean,
    selectedId: number | null,
    hoveredId: number | null,
    focusedIds: Set<number>,
  ) {
    const showEntireGraph = selectedId === VIRTUAL_ROOT_ID;
    for (const [id, view] of this.nodeViews) {
      const visible = !focusActive || showEntireGraph || focusedIds.has(id);
      view.targetOpacity = visible
        ? id === selectedId
          ? 1
          : id === VIRTUAL_ROOT_ID
            ? 0.78
            : id === hoveredId
              ? 1
              : 0.92
        : 0;
    }
    for (const edge of this.edgeViews.values()) {
      edge.targetOpacity = focusActive
        ? showEntireGraph ||
          (focusedIds.has(edge.parentId) && focusedIds.has(edge.childId))
          ? 0.86
          : 0
        : 0.34;
    }
  }

  public updateLabelTargets(
    layout: GraphLayout,
    selectedId: number | null,
    commentNodeId: number | null,
    desiredYaw: number,
    layoutYaw: number,
  ) {
    for (const [id, view] of this.nodeViews) {
      if (!view.label || id === VIRTUAL_ROOT_ID) {
        continue;
      }
      const position = layout.positions.get(id);
      if (!position) {
        continue;
      }
      const moveBehindComments = id === selectedId && commentNodeId === id;
      const labelOffset = moveBehindComments
        ? -LABEL_REAR_OFFSET
        : LABEL_DEFAULT_OFFSET;
      view.labelTargetPosition.set(
        Math.cos(position.angle) * labelOffset,
        1.42,
        Math.sin(position.angle) * labelOffset,
      );
      if (moveBehindComments) {
        // Comment cards extend to screen-right from the focused node. Convert
        // that screen-space left direction into the rotating world's local axes.
        const localYaw = desiredYaw + layoutYaw;
        view.labelTargetPosition.x -=
          Math.cos(localYaw) * LABEL_COMMENT_SIDE_CLEARANCE;
        view.labelTargetPosition.z +=
          Math.sin(localYaw) * LABEL_COMMENT_SIDE_CLEARANCE;
      }
      if (view.label.position.lengthSq() === 0) {
        view.label.position.copy(view.labelTargetPosition);
      }
    }
  }

  public pickNodeAt(
    camera: THREE.PerspectiveCamera,
    pointerNdc: THREE.Vector2,
    viewportHeight: number,
    includeVirtual = false,
  ): number | null {
    this.raycaster.setFromCamera(pointerNdc, camera);
    const ray = this.raycaster.ray;
    const safeHeight = Math.max(1, viewportHeight);
    let nearestId: number | null = null;
    let nearestRayDistance = Number.POSITIVE_INFINITY;

    for (const [id, view] of this.nodeViews) {
      if (id === VIRTUAL_ROOT_ID && !includeVirtual) {
        continue;
      }
      if (
        view.targetOpacity <= 0.01 ||
        view.opacity <= 0.08 ||
        !view.group.visible
      ) {
        continue;
      }

      view.group.getWorldPosition(this.pickCenter);
      this.pickOffset.copy(this.pickCenter).sub(ray.origin);
      if (this.pickOffset.dot(ray.direction) <= 0) {
        continue;
      }

      const rayDistance = ray.distanceToPoint(this.pickCenter);
      const cameraDistance = camera.position.distanceTo(this.pickCenter);
      const worldUnitsPerPixel =
        (2 *
          cameraDistance *
          Math.tan(THREE.MathUtils.degToRad(camera.fov / 2))) /
        safeHeight;
      // The fixed screen-space allowance expands in world units as the camera
      // moves away, keeping distant nodes practical to target.
      const hitDistance =
        view.radius * 1.35 + HOVER_HIT_RADIUS_PX * worldUnitsPerPixel;
      if (rayDistance <= hitDistance && rayDistance < nearestRayDistance) {
        nearestId = id;
        nearestRayDistance = rayDistance;
      }
    }

    return nearestId;
  }

  public getNodeWorldPosition(id: number, out: THREE.Vector3): boolean {
    const view = this.nodeViews.get(id);
    if (!view?.group.visible) {
      return false;
    }
    view.group.updateWorldMatrix(true, false);
    view.group.getWorldPosition(out);

    return true;
  }

  public update(params: {
    delta: number;
    now: number;
    camera: THREE.PerspectiveCamera;
    viewportWidth: number;
    viewportHeight: number;
    layout: GraphLayout;
    zoomRatio: number;
    focusActive: boolean;
    selectedId: number | null;
    hoveredId: number | null;
    focusedIds: Set<number>;
    damping: number;
    opacityDamping: number;
    hoverDamping: number;
  }) {
    const {
      delta,
      now,
      camera,
      viewportWidth,
      viewportHeight,
      layout,
      zoomRatio,
      focusActive,
      selectedId,
      hoveredId,
      focusedIds,
      damping,
      opacityDamping,
      hoverDamping,
    } = params;
    const labelNodeWorldPosition = new THREE.Vector3();
    const safeViewportHeight = Math.max(1, viewportHeight);

    for (const [id, view] of this.nodeViews) {
      const hovered = id === hoveredId;
      const selected = id === selectedId;
      view.group.position.lerp(view.targetPosition, damping);
      view.label?.position.lerp(view.labelTargetPosition, damping);
      view.opacity += (view.targetOpacity - view.opacity) * opacityDamping;
      view.hoverAmount += ((hovered ? 1 : 0) - view.hoverAmount) * hoverDamping;
      const neighborTitle = Boolean(
        focusActive && !selected && focusedIds.has(id),
      );
      const targetLabelScale = selected
        ? 1.1
        : hovered
          ? 1.08
          : neighborTitle
            ? 0.9
            : 1;
      view.labelScale += (targetLabelScale - view.labelScale) * damping;
      const targetLabelEmphasis = hovered ? 1 : neighborTitle ? 0.76 : 1;
      view.labelEmphasis +=
        (targetLabelEmphasis - view.labelEmphasis) * opacityDamping;

      const materialOpacity = view.opacity;
      const hoverAmount = view.hoverAmount;
      view.plate.material.opacity = materialOpacity;
      view.cap.material.opacity = materialOpacity;
      view.crystal.material.opacity = materialOpacity;
      const plateBase = selected
        ? Math.max(view.baseEmissive.plate, 0.68)
        : view.baseEmissive.plate;
      const capBase = selected
        ? Math.max(view.baseEmissive.cap, 0.78)
        : view.baseEmissive.cap;
      const crystalBase = selected
        ? Math.max(view.baseEmissive.crystal, 1.45)
        : view.baseEmissive.crystal;
      view.plate.material.emissiveIntensity = THREE.MathUtils.lerp(
        plateBase,
        Math.max(plateBase + 0.58, 1.18),
        hoverAmount,
      );
      view.cap.material.emissiveIntensity = THREE.MathUtils.lerp(
        capBase,
        Math.max(capBase + 0.5, 1.18),
        hoverAmount,
      );
      view.crystal.material.emissiveIntensity = THREE.MathUtils.lerp(
        crystalBase,
        Math.max(crystalBase + 0.68, 2.25),
        hoverAmount,
      );
      const innerRingBase = materialOpacity * (selected ? 0.9 : 0.52);
      const outerRingBase = materialOpacity * (selected ? 0.95 : 0.48);
      view.innerRing.material.opacity = THREE.MathUtils.lerp(
        innerRingBase,
        materialOpacity,
        hoverAmount * 0.72,
      );
      view.outerRing.material.opacity = THREE.MathUtils.lerp(
        outerRingBase,
        materialOpacity,
        hoverAmount * 0.82,
      );
      const glowBase = materialOpacity * (selected ? 0.95 : 0.38);
      view.glow.material.opacity = THREE.MathUtils.lerp(
        glowBase,
        materialOpacity,
        hoverAmount,
      );

      const zoom = THREE.MathUtils.clamp(zoomRatio, 0.45, 5);
      const labelScreenHeight = THREE.MathUtils.clamp(
        36 * zoom ** 0.32,
        34,
        46,
      );
      view.group.getWorldPosition(labelNodeWorldPosition);
      const nodeDistance = Math.max(
        1,
        camera.position.distanceTo(labelNodeWorldPosition),
      );
      const worldPerPixel =
        (2 *
          nodeDistance *
          Math.tan(THREE.MathUtils.degToRad(camera.fov / 2))) /
        safeViewportHeight;
      const rootScale = id === VIRTUAL_ROOT_ID ? 1.08 : 1;
      const labelWorldHeight =
        labelScreenHeight * worldPerPixel * view.labelScale * rootScale;
      view.label?.scale.set(
        labelWorldHeight * LABEL_ASPECT_RATIO,
        labelWorldHeight,
        1,
      );

      const pulse =
        (selected ? 1.08 + Math.sin(now * 0.004) * 0.035 : 1) *
        (1 + hoverAmount * 0.055);
      view.plate.scale.setScalar(view.radius * pulse);
      view.plate.scale.y = (id === VIRTUAL_ROOT_ID ? 1.5 : 1) * pulse;
      view.cap.scale.setScalar(view.radius * pulse);
      view.cap.scale.y = (id === VIRTUAL_ROOT_ID ? 1.45 : 1) * pulse;
      view.crystal.rotation.y += delta * (selected ? 0.75 : 0.24);
      view.crystal.scale.setScalar(
        view.radius * (id === VIRTUAL_ROOT_ID ? 1.18 : 0.9) * pulse,
      );
      view.innerRing.scale.setScalar(view.radius * pulse);
      view.outerRing.rotation.z += delta * (selected ? 0.35 : 0.09);
      view.outerRing.scale.setScalar(view.radius * pulse);
      view.glow.scale.setScalar(
        view.radius *
          (id === VIRTUAL_ROOT_ID ? 4.8 : 4.1) *
          (1 + hoverAmount * 0.2),
      );
      this.updateHoverParticles(view, now, hoverAmount);
      view.group.visible = view.opacity > 0.01;
    }

    for (const edge of this.edgeViews.values()) {
      const material = edge.line.material;
      material.opacity +=
        (edge.targetOpacity - material.opacity) * opacityDamping;
      edge.line.visible = material.opacity > 0.01;
    }

    this.updateLabelDeclutter(
      camera,
      Math.max(1, viewportWidth),
      safeViewportHeight,
      layout,
      zoomRatio,
      selectedId,
      hoveredId,
    );
    for (const view of this.nodeViews.values()) {
      view.labelOpacity +=
        (view.targetLabelOpacity - view.labelOpacity) * opacityDamping;
      if (view.label) {
        view.label.material.opacity = view.labelOpacity;
      }
    }
  }

  public dispose() {
    for (const id of this.nodeViews.keys()) {
      this.removeNode(id);
    }
    for (const edge of this.edgeViews.values()) {
      this.disposeEdge(edge);
    }
    this.edgeViews.clear();
    this.plateGeometry.dispose();
    this.capGeometry.dispose();
    this.crystalGeometry.dispose();
    this.innerRingGeometry.dispose();
    this.outerRingGeometry.dispose();
    this.glowTexture.dispose();
  }

  private createNode(
    node: GraphNode,
    layout: GraphLayout,
    theme: ThemeMode,
  ): NodeVisual {
    const virtual = node.id === VIRTUAL_ROOT_ID;
    const palette = SCENE_PALETTES[theme];
    const radius = virtual ? 1.35 : 0.72;
    const accent = nodeAccentColor(theme, virtual, node.completedAt != null);
    const basePlateEmissive = virtual ? 0.32 : 0.12;
    const baseCapEmissive = virtual ? 0.55 : 0.32;
    const baseCrystalEmissive = virtual ? 1.9 : 1.1;
    const plateMaterial = new THREE.MeshStandardMaterial({
      color: virtual ? palette.virtualPlate : palette.plate,
      emissive: accent,
      emissiveIntensity: basePlateEmissive,
      roughness: 0.32,
      metalness: 0.72,
      transparent: true,
      opacity: 0,
    });
    const capMaterial = new THREE.MeshStandardMaterial({
      color: accent,
      emissive: accent,
      emissiveIntensity: baseCapEmissive,
      roughness: 0.24,
      metalness: 0.66,
      transparent: true,
      opacity: 0,
    });
    const crystalMaterial = new THREE.MeshStandardMaterial({
      color: virtual ? palette.virtualCrystal : palette.crystal,
      emissive: accent,
      emissiveIntensity: baseCrystalEmissive,
      roughness: 0.1,
      metalness: 0.32,
      transparent: true,
      opacity: 0,
    });
    const lineMaterial = new THREE.MeshBasicMaterial({
      color: accent,
      transparent: true,
      opacity: 0,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
      side: THREE.DoubleSide,
    });

    const group = new THREE.Group();
    const plate = new THREE.Mesh(this.plateGeometry, plateMaterial);
    plate.scale.set(radius, virtual ? 1.5 : 1, radius);
    plate.rotation.y = Math.PI / 6;
    plate.userData.nodeId = node.id;

    const cap = new THREE.Mesh(this.capGeometry, capMaterial);
    cap.scale.set(radius, virtual ? 1.45 : 1, radius);
    cap.position.y = 0.2;
    cap.rotation.y = Math.PI / 6;

    const crystal = new THREE.Mesh(this.crystalGeometry, crystalMaterial);
    crystal.scale.setScalar(radius * (virtual ? 1.18 : 0.9));
    crystal.position.y = virtual ? 0.72 : 0.48;
    crystal.rotation.y = Math.PI / 4;

    const innerRing = new THREE.Mesh(this.innerRingGeometry, lineMaterial);
    innerRing.scale.setScalar(radius);
    innerRing.rotation.x = -Math.PI / 2;
    innerRing.position.y = 0.17;

    const outerRing = new THREE.Mesh(
      this.outerRingGeometry,
      lineMaterial.clone(),
    );
    outerRing.scale.setScalar(radius);
    outerRing.rotation.x = -Math.PI / 2;
    outerRing.rotation.z = virtual ? 0.4 : Math.PI / 6;
    outerRing.position.y = 0.205;

    const glow = new THREE.Sprite(
      new THREE.SpriteMaterial({
        map: this.glowTexture,
        color: accent,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      }),
    );
    glow.scale.setScalar(radius * (virtual ? 4.8 : 4.1));
    glow.position.y = 0.28;

    const particlePositions = new Float32Array(HOVER_PARTICLE_COUNT * 3);
    const particleSeeds = new Float32Array(
      HOVER_PARTICLE_COUNT * HOVER_PARTICLE_STRIDE,
    );
    for (let index = 0; index < HOVER_PARTICLE_COUNT; index++) {
      const offset = index * HOVER_PARTICLE_STRIDE;
      particleSeeds[offset] = Math.random() * Math.PI * 2;
      particleSeeds[offset + 1] = 0.8 + Math.random() * 1.35;
      particleSeeds[offset + 2] = 0.7 + Math.random() * 0.85;
      particleSeeds[offset + 3] = Math.random();
      particleSeeds[offset + 4] = 1.35 + Math.random() * 1.25;
    }
    const particleGeometry = new THREE.BufferGeometry();
    particleGeometry.setAttribute(
      "position",
      new THREE.BufferAttribute(particlePositions, 3),
    );
    const particles = new THREE.Points(
      particleGeometry,
      new THREE.PointsMaterial({
        map: this.glowTexture,
        color: accent,
        size: radius * 0.34,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        sizeAttenuation: true,
      }),
    );
    particles.frustumCulled = false;
    particles.visible = false;

    const label = virtual
      ? null
      : new THREE.Sprite(
          new THREE.SpriteMaterial({
            map: createLabelTexture(
              node.title,
              node.completedAt != null,
              false,
              theme,
            ),
            transparent: true,
            opacity: 0,
            depthWrite: false,
            toneMapped: false,
          }),
        );
    if (label) {
      label.center.set(0.5, 0);
      label.scale.set(4.8, 4.8 / LABEL_ASPECT_RATIO, 1);
    }
    const labelTargetPosition = new THREE.Vector3();

    group.add(plate, cap, crystal, innerRing, outerRing, glow, particles);
    if (label) {
      group.add(label);
    }
    const position = layout.positions.get(node.id);
    group.position.set(position?.x ?? 0, 0.9, position?.z ?? 0);

    return {
      group,
      plate,
      cap,
      crystal,
      innerRing,
      outerRing,
      glow,
      particles,
      particleSeeds,
      label,
      labelTargetPosition,
      radius,
      accent,
      title: node.title,
      completed: node.completedAt != null,
      baseEmissive: {
        plate: basePlateEmissive,
        cap: baseCapEmissive,
        crystal: baseCrystalEmissive,
      },
      targetPosition: group.position.clone(),
      targetOpacity: 0,
      opacity: 0,
      hoverAmount: 0,
      labelScale: 1,
      labelEmphasis: 1,
      labelOpacity: 0,
      targetLabelOpacity: 0,
    };
  }

  private updateNodeAppearance(
    view: NodeVisual,
    node: GraphNode,
    theme: ThemeMode,
  ) {
    const virtual = node.id === VIRTUAL_ROOT_ID;
    const completed = node.completedAt != null;
    const accent = nodeAccentColor(theme, virtual, completed);
    view.accent = accent;
    view.title = node.title;
    view.completed = completed;
    this.applyAccentColor(view, accent);
    if (!view.label || virtual) {
      return;
    }
    this.refreshLabelTexture(view, node.title, completed, theme);
  }

  private applyAccentColor(view: NodeVisual, accent: number) {
    view.plate.material.emissive.setHex(accent);
    view.cap.material.color.setHex(accent);
    view.cap.material.emissive.setHex(accent);
    view.crystal.material.emissive.setHex(accent);
    view.innerRing.material.color.setHex(accent);
    view.outerRing.material.color.setHex(accent);
    view.glow.material.color.setHex(accent);
    view.particles.material.color.setHex(accent);
  }

  private refreshLabelTexture(
    view: NodeVisual,
    title: string,
    completed: boolean,
    theme: ThemeMode,
  ) {
    if (!view.label) {
      return;
    }
    const previousMap = view.label.material.map;
    view.label.material.map = createLabelTexture(
      title,
      completed,
      false,
      theme,
    );
    view.label.material.needsUpdate = true;
    previousMap?.dispose();
  }

  private removeNode(id: number) {
    const view = this.nodeViews.get(id);
    if (!view) {
      return;
    }
    this.world.remove(view.group);
    view.label?.material.map?.dispose();
    view.label?.material.dispose();
    view.plate.material.dispose();
    view.cap.material.dispose();
    view.crystal.material.dispose();
    view.innerRing.material.dispose();
    view.outerRing.material.dispose();
    view.glow.material.dispose();
    view.particles.geometry.dispose();
    view.particles.material.dispose();
    this.nodeViews.delete(id);
  }

  private rebuildEdges(layout: GraphLayout, theme: ThemeMode) {
    const palette = SCENE_PALETTES[theme];
    for (const [id, edge] of this.edgeViews) {
      this.world.remove(edge.line);
      this.disposeEdge(edge);
      this.edgeViews.delete(id);
    }
    for (const edge of layout.edges) {
      const parent = layout.positions.get(edge.parentId);
      const child = layout.positions.get(edge.childId);
      if (!parent || !child) {
        continue;
      }
      const material = new THREE.LineBasicMaterial({
        color: palette.edge,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      });
      const geometry = createEdgeGeometry(
        new THREE.Vector3(parent.x, 0, parent.z),
        new THREE.Vector3(child.x, 0, child.z),
      );
      const line = new THREE.Line(geometry, material);
      this.world.add(line);
      this.edgeViews.set(edge.id, {
        line,
        parentId: edge.parentId,
        childId: edge.childId,
        targetOpacity: 0.34,
      });
    }
  }

  private disposeEdge(edge: EdgeVisual) {
    edge.line.geometry.dispose();
    edge.line.material.dispose();
  }

  private updateLabelDeclutter(
    camera: THREE.PerspectiveCamera,
    width: number,
    height: number,
    layout: GraphLayout,
    zoomRatio: number,
    selectedId: number | null,
    hoveredId: number | null,
  ) {
    const cameraRight = new THREE.Vector3(1, 0, 0).applyQuaternion(
      camera.quaternion,
    );
    const cameraUp = new THREE.Vector3(0, 1, 0).applyQuaternion(
      camera.quaternion,
    );
    const cameraForward = new THREE.Vector3(0, 0, -1).applyQuaternion(
      camera.quaternion,
    );
    const labelCenter = new THREE.Vector3();
    const projected = new THREE.Vector3();
    const candidates: LabelCandidate[] = [];
    const hideOverviewLabels = zoomRatio < LABEL_ZOOM_HIDE_THRESHOLD;

    for (const [id, view] of this.nodeViews) {
      if (view.opacity < 0.05 || !view.group.visible || !view.label) {
        view.targetLabelOpacity = 0;

        continue;
      }
      if (hideOverviewLabels && id !== hoveredId) {
        view.targetLabelOpacity = 0;

        continue;
      }
      view.label.getWorldPosition(labelCenter);
      // Labels are anchored at their lower edge so they grow upward from the
      // node. Collision rectangles need the visual center instead.
      labelCenter.addScaledVector(cameraUp, view.label.scale.y * 0.5);
      if (labelCenter.clone().sub(camera.position).dot(cameraForward) <= 0.1) {
        view.targetLabelOpacity = 0;

        continue;
      }

      const halfWidth = view.label.scale.x * 0.5;
      const halfHeight = view.label.scale.y * cameraUp.length() * 0.5;
      const points = [
        labelCenter
          .clone()
          .addScaledVector(cameraRight, -halfWidth)
          .addScaledVector(cameraUp, halfHeight),
        labelCenter
          .clone()
          .addScaledVector(cameraRight, halfWidth)
          .addScaledVector(cameraUp, halfHeight),
        labelCenter
          .clone()
          .addScaledVector(cameraRight, halfWidth)
          .addScaledVector(cameraUp, -halfHeight),
        labelCenter
          .clone()
          .addScaledVector(cameraRight, -halfWidth)
          .addScaledVector(cameraUp, -halfHeight),
      ].map((point) => {
        projected.copy(point).project(camera);

        return {
          x: (projected.x * 0.5 + 0.5) * width,
          y: (-projected.y * 0.5 + 0.5) * height,
        };
      });
      const left = Math.min(...points.map((point) => point.x));
      const right = Math.max(...points.map((point) => point.x));
      const top = Math.min(...points.map((point) => point.y));
      const bottom = Math.max(...points.map((point) => point.y));
      const depth = layout.positions.get(id)?.depth ?? Number.MAX_SAFE_INTEGER;
      candidates.push({
        id,
        view,
        depth,
        left,
        right,
        top,
        bottom,
        readable: bottom - top >= 12,
      });
    }

    candidates.sort((a, b) => {
      const aSelected = a.id === selectedId;
      const bSelected = b.id === selectedId;
      if (aSelected !== bSelected) {
        return aSelected ? -1 : 1;
      }
      const aHovered = a.id === hoveredId;
      const bHovered = b.id === hoveredId;
      if (aHovered !== bHovered) {
        return aHovered ? -1 : 1;
      }
      if (a.depth !== b.depth) {
        return a.depth - b.depth;
      }

      return a.top - b.top;
    });
    const occupied: LabelCandidate[] = [];
    for (const candidate of candidates) {
      const selected = candidate.id === selectedId;
      const hovered = candidate.id === hoveredId;
      const offscreen =
        candidate.right < 0 ||
        candidate.left > width ||
        candidate.bottom < 0 ||
        candidate.top > height;
      const collides = occupied.some(
        (other) =>
          candidate.left < other.right + LABEL_GAP &&
          candidate.right + LABEL_GAP > other.left &&
          candidate.top < other.bottom + LABEL_GAP &&
          candidate.bottom + LABEL_GAP > other.top,
      );
      const visible =
        selected ||
        (hovered && !collides) ||
        (!offscreen && candidate.readable && !collides);
      candidate.view.targetLabelOpacity = visible
        ? candidate.view.opacity * candidate.view.labelEmphasis
        : 0;
      if (visible) {
        occupied.push(candidate);
      }
    }
  }

  private updateHoverParticles(
    view: NodeVisual,
    now: number,
    hoverAmount: number,
  ) {
    if (hoverAmount <= 0.001) {
      view.particles.material.opacity = 0;
      view.particles.visible = false;

      return;
    }
    const position = view.particles.geometry.getAttribute(
      "position",
    ) as THREE.BufferAttribute;
    const positions = position.array as Float32Array;
    for (let index = 0; index < HOVER_PARTICLE_COUNT; index++) {
      const seedOffset = index * HOVER_PARTICLE_STRIDE;
      const positionOffset = index * 3;
      const angle = view.particleSeeds[seedOffset];
      const orbitFactor = view.particleSeeds[seedOffset + 1];
      const speed = view.particleSeeds[seedOffset + 2];
      const phase = view.particleSeeds[seedOffset + 3];
      const rise = view.particleSeeds[seedOffset + 4];
      const progress = (phase + now * 0.0001 * speed) % 1;
      const radius = view.radius * orbitFactor * (1 - progress * 0.24);
      positions[positionOffset] = Math.cos(angle + progress * 1.15) * radius;
      positions[positionOffset + 1] =
        0.18 +
        progress * rise +
        Math.sin((progress + phase) * Math.PI * 2) * 0.07;
      positions[positionOffset + 2] =
        Math.sin(angle + progress * 1.15) * radius;
    }
    position.needsUpdate = true;
    view.particles.material.opacity = hoverAmount * 0.86;
    view.particles.material.size = view.radius * (0.24 + hoverAmount * 0.1);
    view.particles.visible = hoverAmount > 0.01;
  }
}
