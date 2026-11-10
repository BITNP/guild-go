import * as THREE from "three";

import type { GraphLayout } from "../layout";
import { VIRTUAL_ROOT_ID } from "../layout";
import { LABEL_ASPECT_RATIO } from "../visuals";
import { updateLabelVisibility } from "./labelLayout";
import type { NodeVisual } from "./nodeVisual";
import {
  HOVER_PARTICLE_COUNT,
  HOVER_PARTICLE_STRIDE,
  applyNodeStyle,
  createNodeVisual,
  createNodeVisualResources,
  disposeNodeVisual,
  disposeNodeVisualResources,
} from "./nodeVisual";

const LABEL_REAR_OFFSET = 2.15;
const LABEL_DEFAULT_OFFSET = 1.7;
const LABEL_COMMENT_SIDE_CLEARANCE = (4.8 * 1.1) / 2;
const HOVER_HIT_RADIUS_PX = 24;
const HOVER_FADE_SPEED = 9;

export interface NodeLayerState {
  commentNodeId: number | null;
  focusActive: boolean;
  focusedIds: ReadonlySet<number>;
  hoveredId: number | null;
  selectedId: number | null;
}

interface NodeLayerUpdate {
  camera: THREE.Camera;
  cameraDistance: number;
  delta: number;
  fitDistance: number;
  now: number;
  viewportHeight: number;
  viewportWidth: number;
}

export class NodeLayer {
  private readonly world: THREE.Group;
  private readonly resources = createNodeVisualResources();
  private readonly views = new Map<number, NodeVisual>();
  private readonly nodes = new Map<number, GraphNode>();
  private readonly pickCenter = new THREE.Vector3();
  private readonly pickOffset = new THREE.Vector3();
  private readonly labelNodeWorldPosition = new THREE.Vector3();
  private layout: GraphLayout | null = null;
  private theme: ThemeMode;
  private state: NodeLayerState = {
    commentNodeId: null,
    focusActive: false,
    focusedIds: new Set(),
    hoveredId: null,
    selectedId: null,
  };

  constructor(world: THREE.Group, theme: ThemeMode) {
    this.world = world;
    this.theme = theme;
  }

  public sync(nodes: GraphNode[], layout: GraphLayout) {
    this.layout = layout;
    this.nodes.clear();

    const activeIds = new Set<number>();
    for (const node of nodes) {
      activeIds.add(node.id);
      this.nodes.set(node.id, node);
    }

    for (const id of this.views.keys()) {
      if (activeIds.has(id)) {
        continue;
      }
      this.removeVisual(id);
    }

    for (const node of nodes) {
      const position = layout.positions.get(node.id);
      if (!position) {
        continue;
      }

      let view = this.views.get(node.id);
      if (!view) {
        view = createNodeVisual(
          node,
          position.x,
          position.z,
          this.theme,
          this.resources,
        );
        this.views.set(node.id, view);
        this.world.add(view.group);
      } else if (
        view.completed !== (node.completedAt != null) ||
        view.title !== node.title
      ) {
        applyNodeStyle(view, node, this.theme);
      }

      view.targetPosition.set(position.x, 0.9, position.z);
    }
  }

  public setTheme(theme: ThemeMode) {
    this.theme = theme;
    for (const [id, view] of this.views) {
      const node = this.nodes.get(id);
      if (node) {
        applyNodeStyle(view, node, theme);
      }
    }
  }

  public setState(state: NodeLayerState) {
    this.state = state;
  }

  public refreshLabelTargets(desiredYaw: number, layoutYaw: number) {
    if (!this.layout) {
      return;
    }

    for (const [id, view] of this.views) {
      if (!view.label || id === VIRTUAL_ROOT_ID) {
        continue;
      }

      const position = this.layout.positions.get(id);
      if (!position) {
        continue;
      }

      const moveBehindComments =
        id === this.state.selectedId && this.state.commentNodeId === id;
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

  public applyVisualState() {
    const showEntireGraph = this.state.selectedId === VIRTUAL_ROOT_ID;

    for (const [id, view] of this.views) {
      const visible =
        !this.state.focusActive ||
        showEntireGraph ||
        this.state.focusedIds.has(id);

      view.targetOpacity = visible
        ? id === this.state.selectedId
          ? 1
          : id === VIRTUAL_ROOT_ID
            ? 0.78
            : id === this.state.hoveredId
              ? 1
              : 0.92
        : 0;
    }
  }

  public getWorldPosition(id: number, target: THREE.Vector3): boolean {
    const position = this.layout?.positions.get(id);
    if (!position) {
      return false;
    }

    target
      .set(position.x, 0.9, position.z)
      .applyMatrix4(this.world.matrixWorld);

    return true;
  }

  public getVisualWorldPosition(id: number, target: THREE.Vector3): boolean {
    const view = this.views.get(id);
    if (!view?.group.visible) {
      return false;
    }

    view.group.getWorldPosition(target);

    return true;
  }

  public pick(
    raycaster: THREE.Raycaster,
    camera: THREE.Camera,
    viewportHeight: number,
    includeVirtual: boolean,
  ): number | null {
    const ray = raycaster.ray;
    let nearestId: number | null = null;
    let nearestRayDistance = Number.POSITIVE_INFINITY;

    for (const [id, view] of this.views) {
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
          Math.tan(THREE.MathUtils.degToRad(this.cameraFov(camera) / 2))) /
        Math.max(1, viewportHeight);
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

  public update(delta: number, update: NodeLayerUpdate) {
    const layout = this.layout;
    if (!layout) {
      return;
    }

    const damping = 1 - Math.exp(-delta * 5.2);
    const opacityDamping = 1 - Math.exp(-delta * 8);
    const hoverDamping = 1 - Math.exp(-delta * HOVER_FADE_SPEED);

    for (const [id, view] of this.views) {
      const hovered = id === this.state.hoveredId;
      const selected = id === this.state.selectedId;

      view.group.position.lerp(view.targetPosition, damping);
      view.label?.position.lerp(view.labelTargetPosition, damping);
      view.opacity += (view.targetOpacity - view.opacity) * opacityDamping;
      view.hoverAmount += ((hovered ? 1 : 0) - view.hoverAmount) * hoverDamping;

      const neighborTitle = Boolean(
        this.state.focusActive && !selected && this.state.focusedIds.has(id),
      );
      const targetLabelScale = selected
        ? 1.1
        : hovered
          ? 1.08
          : neighborTitle
            ? 0.9
            : 1;
      const targetLabelEmphasis = hovered ? 1 : neighborTitle ? 0.76 : 1;

      view.labelScale += (targetLabelScale - view.labelScale) * damping;
      view.labelEmphasis +=
        (targetLabelEmphasis - view.labelEmphasis) * opacityDamping;

      this.updateNodeMaterials(view, selected);
      this.updateNodeGeometry(view, id, selected, update);
      this.updateHoverParticles(view, update.now, view.hoverAmount);
      view.group.visible = view.opacity > 0.01;
    }

    updateLabelVisibility(
      {
        camera: update.camera,
        height: update.viewportHeight,
        hoveredId: this.state.hoveredId,
        layout,
        selectedId: this.state.selectedId,
        views: this.views,
        width: update.viewportWidth,
      },
      update.fitDistance / Math.max(update.cameraDistance, 1),
    );

    for (const view of this.views.values()) {
      view.labelOpacity +=
        (view.targetLabelOpacity - view.labelOpacity) * opacityDamping;

      if (view.label) {
        view.label.material.opacity = view.labelOpacity;
      }
    }
  }

  public dispose() {
    for (const id of this.views.keys()) {
      this.removeVisual(id);
    }
    disposeNodeVisualResources(this.resources);
  }

  private updateNodeMaterials(view: NodeVisual, selected: boolean) {
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
  }

  private updateNodeGeometry(
    view: NodeVisual,
    id: number,
    selected: boolean,
    update: NodeLayerUpdate,
  ) {
    const zoom = THREE.MathUtils.clamp(
      update.fitDistance / Math.max(update.cameraDistance, 1),
      0.45,
      5,
    );
    const labelScreenHeight = THREE.MathUtils.clamp(48 * zoom ** 0.32, 46, 62);
    const viewportHeight = Math.max(1, update.viewportHeight);

    view.group.getWorldPosition(this.labelNodeWorldPosition);

    const nodeDistance = Math.max(
      1,
      update.camera.position.distanceTo(this.labelNodeWorldPosition),
    );
    const worldPerPixel =
      (2 *
        nodeDistance *
        Math.tan(THREE.MathUtils.degToRad(this.cameraFov(update.camera) / 2))) /
      viewportHeight;
    const rootScale = id === VIRTUAL_ROOT_ID ? 1.08 : 1;
    const labelWorldHeight =
      labelScreenHeight * worldPerPixel * view.labelScale * rootScale;

    view.label?.scale.set(
      labelWorldHeight * LABEL_ASPECT_RATIO,
      labelWorldHeight,
      1,
    );

    const pulse =
      (selected ? 1.08 + Math.sin(update.now * 0.004) * 0.035 : 1) *
      (1 + view.hoverAmount * 0.055);

    view.plate.scale.setScalar(view.radius * pulse);
    view.plate.scale.y = (id === VIRTUAL_ROOT_ID ? 1.5 : 1) * pulse;
    view.cap.scale.setScalar(view.radius * pulse);
    view.cap.scale.y = (id === VIRTUAL_ROOT_ID ? 1.45 : 1) * pulse;
    view.crystal.rotation.y +=
      update.delta * (id === this.state.selectedId ? 0.75 : 0.24);
    view.crystal.scale.setScalar(
      view.radius * (id === VIRTUAL_ROOT_ID ? 1.18 : 0.9) * pulse,
    );
    view.innerRing.scale.setScalar(view.radius * pulse);
    view.outerRing.rotation.z +=
      update.delta * (id === this.state.selectedId ? 0.35 : 0.09);
    view.outerRing.scale.setScalar(view.radius * pulse);
    view.glow.scale.setScalar(
      view.radius *
        (id === VIRTUAL_ROOT_ID ? 4.8 : 4.1) *
        (1 + view.hoverAmount * 0.2),
    );
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

  private removeVisual(id: number) {
    const view = this.views.get(id);
    if (!view) {
      return;
    }

    this.world.remove(view.group);
    disposeNodeVisual(view);
    this.views.delete(id);
  }

  private cameraFov(camera: THREE.Camera) {
    return camera instanceof THREE.PerspectiveCamera ? camera.fov : 44;
  }
}
