import * as THREE from "three";

import type { GraphLayout } from "../layout";
import type { NodeVisual } from "./nodeVisual";

const LABEL_GAP = 10;
const LABEL_ZOOM_HIDE_THRESHOLD = 0.68;

interface LabelCandidate {
  bottom: number;
  depth: number;
  id: number;
  left: number;
  readable: boolean;
  right: number;
  top: number;
  view: NodeVisual;
}

interface LabelVisibilityOptions {
  camera: THREE.Camera;
  height: number;
  hoveredId: number | null;
  layout: GraphLayout;
  selectedId: number | null;
  views: Map<number, NodeVisual>;
  width: number;
}

export function updateLabelVisibility(
  options: LabelVisibilityOptions,
  zoomRatio: number,
) {
  const width = Math.max(1, options.width);
  const height = Math.max(1, options.height);
  const selectedId = options.selectedId;
  const hoveredId = options.hoveredId;
  const cameraRight = new THREE.Vector3(1, 0, 0).applyQuaternion(
    options.camera.quaternion,
  );
  const cameraUp = new THREE.Vector3(0, 1, 0).applyQuaternion(
    options.camera.quaternion,
  );
  const cameraForward = new THREE.Vector3(0, 0, -1).applyQuaternion(
    options.camera.quaternion,
  );
  const labelCenter = new THREE.Vector3();
  const projected = new THREE.Vector3();
  const candidates: LabelCandidate[] = [];
  const hideOverviewLabels = zoomRatio < LABEL_ZOOM_HIDE_THRESHOLD;

  for (const [id, view] of options.views) {
    if (view.opacity < 0.05 || !view.group.visible) {
      view.targetLabelOpacity = 0;

      continue;
    }
    if (!view.label) {
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

    const towardLabel = labelCenter.clone().sub(options.camera.position);
    if (towardLabel.dot(cameraForward) <= 0.1) {
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
      projected.copy(point).project(options.camera);

      return {
        x: (projected.x * 0.5 + 0.5) * width,
        y: (-projected.y * 0.5 + 0.5) * height,
      };
    });
    const left = Math.min(...points.map((point) => point.x));
    const right = Math.max(...points.map((point) => point.x));
    const top = Math.min(...points.map((point) => point.y));
    const bottom = Math.max(...points.map((point) => point.y));

    candidates.push({
      id,
      view,
      depth: options.layout.positions.get(id)?.depth ?? Number.MAX_SAFE_INTEGER,
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
