import * as THREE from "three";

import type { ThemeMode } from "@/composables/useTheme";

import type { GraphLayout } from "../layout";
import { VIRTUAL_ROOT_ID } from "../layout";
import { createEdgeGeometry } from "../visuals";
import { SCENE_PALETTES } from "./palette";

interface EdgeVisual {
  childId: number;
  line: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>;
  parentId: number;
  targetOpacity: number;
}

export interface EdgeLayerState {
  focusActive: boolean;
  focusedIds: ReadonlySet<number>;
  selectedId: number | null;
}

export class EdgeLayer {
  private readonly world: THREE.Group;
  private readonly views = new Map<string, EdgeVisual>();
  private theme: ThemeMode;
  private state: EdgeLayerState = {
    focusActive: false,
    focusedIds: new Set(),
    selectedId: null,
  };

  constructor(world: THREE.Group, theme: ThemeMode) {
    this.world = world;
    this.theme = theme;
  }

  public sync(layout: GraphLayout) {
    this.clear();

    for (const edge of layout.edges) {
      const parent = layout.positions.get(edge.parentId);
      const child = layout.positions.get(edge.childId);
      if (!parent || !child) {
        continue;
      }

      const material = new THREE.LineBasicMaterial({
        color: SCENE_PALETTES[this.theme].edge,
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
      this.views.set(edge.id, {
        line,
        parentId: edge.parentId,
        childId: edge.childId,
        targetOpacity: 0.34,
      });
    }
  }

  public setTheme(theme: ThemeMode) {
    this.theme = theme;
    for (const edge of this.views.values()) {
      edge.line.material.color.setHex(SCENE_PALETTES[theme].edge);
    }
  }

  public setState(state: EdgeLayerState) {
    this.state = state;
  }

  public applyVisualState() {
    const showEntireGraph = this.state.selectedId === VIRTUAL_ROOT_ID;

    for (const edge of this.views.values()) {
      edge.targetOpacity = this.state.focusActive
        ? showEntireGraph ||
          (this.state.focusedIds.has(edge.parentId) &&
            this.state.focusedIds.has(edge.childId))
          ? 0.86
          : 0
        : 0.34;
    }
  }

  public update(delta: number) {
    const opacityDamping = 1 - Math.exp(-delta * 8);

    for (const edge of this.views.values()) {
      edge.line.material.opacity +=
        (edge.targetOpacity - edge.line.material.opacity) * opacityDamping;
      edge.line.visible = edge.line.material.opacity > 0.01;
    }
  }

  public dispose() {
    this.clear();
  }

  private clear() {
    for (const edge of this.views.values()) {
      this.world.remove(edge.line);
      edge.line.geometry.dispose();
      edge.line.material.dispose();
    }
    this.views.clear();
  }
}
