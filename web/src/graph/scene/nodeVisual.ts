import * as THREE from "three";

import { VIRTUAL_ROOT_ID } from "../layout";
import { createLabelTexture, createNodeGlowTexture } from "../visuals";
import { SCENE_PALETTES, nodeAccent } from "./palette";

export const HOVER_PARTICLE_COUNT = 18;
export const HOVER_PARTICLE_STRIDE = 5;

interface NodeVisualResources {
  capGeometry: THREE.CylinderGeometry;
  crystalGeometry: THREE.OctahedronGeometry;
  glowTexture: THREE.CanvasTexture;
  innerRingGeometry: THREE.RingGeometry;
  outerRingGeometry: THREE.RingGeometry;
  plateGeometry: THREE.CylinderGeometry;
}

export interface NodeVisual {
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

export const createNodeVisualResources = (): NodeVisualResources => ({
  plateGeometry: new THREE.CylinderGeometry(1, 1, 0.16, 6),
  capGeometry: new THREE.CylinderGeometry(0.72, 0.72, 0.3, 6),
  crystalGeometry: new THREE.OctahedronGeometry(0.52, 0),
  innerRingGeometry: new THREE.RingGeometry(1.06, 1.15, 64),
  outerRingGeometry: new THREE.RingGeometry(
    1.36,
    1.41,
    64,
    1,
    0.22,
    Math.PI * 1.31,
  ),
  glowTexture: createNodeGlowTexture(),
});

export function createNodeVisual(
  node: GraphNode,
  x: number,
  z: number,
  theme: ThemeMode,
  resources: NodeVisualResources,
): NodeVisual {
  const palette = SCENE_PALETTES[theme];
  const virtual = node.id === VIRTUAL_ROOT_ID;
  const completed = node.completedAt != null;
  const radius = virtual ? 1.35 : 0.72;
  const accent = nodeAccent(theme, completed, virtual);
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
  const ringMaterial = new THREE.MeshBasicMaterial({
    color: accent,
    transparent: true,
    opacity: 0,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    side: THREE.DoubleSide,
  });

  const group = new THREE.Group();
  const plate = new THREE.Mesh(resources.plateGeometry, plateMaterial);
  plate.scale.set(radius, virtual ? 1.5 : 1, radius);
  plate.rotation.y = Math.PI / 6;
  plate.userData.nodeId = node.id;

  const cap = new THREE.Mesh(resources.capGeometry, capMaterial);
  cap.scale.set(radius, virtual ? 1.45 : 1, radius);
  cap.position.y = 0.2;
  cap.rotation.y = Math.PI / 6;

  const crystal = new THREE.Mesh(resources.crystalGeometry, crystalMaterial);
  crystal.scale.setScalar(radius * (virtual ? 1.18 : 0.9));
  crystal.position.y = virtual ? 0.72 : 0.48;
  crystal.rotation.y = Math.PI / 4;

  const innerRing = new THREE.Mesh(resources.innerRingGeometry, ringMaterial);
  innerRing.scale.setScalar(radius);
  innerRing.rotation.x = -Math.PI / 2;
  innerRing.position.y = 0.17;

  const outerRing = new THREE.Mesh(
    resources.outerRingGeometry,
    ringMaterial.clone(),
  );
  outerRing.scale.setScalar(radius);
  outerRing.rotation.x = -Math.PI / 2;
  outerRing.rotation.z = virtual ? 0.4 : Math.PI / 6;
  outerRing.position.y = 0.205;

  const glow = new THREE.Sprite(
    new THREE.SpriteMaterial({
      map: resources.glowTexture,
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
      map: resources.glowTexture,
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
          map: createLabelTexture(node.title, completed, false, theme),
          transparent: true,
          opacity: 0,
          depthWrite: false,
          toneMapped: false,
        }),
      );
  if (label) {
    label.center.set(0.5, 0);
    label.scale.set(4.8, 0.9, 1);
  }

  const labelTargetPosition = new THREE.Vector3();
  group.position.set(x, 0.9, z);
  group.add(plate, cap, crystal, innerRing, outerRing, glow, particles);
  if (label) {
    group.add(label);
  }

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
    completed,
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

export function applyNodeStyle(
  view: NodeVisual,
  node: GraphNode,
  theme: ThemeMode,
) {
  const palette = SCENE_PALETTES[theme];
  const virtual = node.id === VIRTUAL_ROOT_ID;
  const completed = node.completedAt != null;
  const accent = nodeAccent(theme, completed, virtual);

  view.accent = accent;
  view.title = node.title;
  view.completed = completed;
  view.plate.material.color.setHex(
    virtual ? palette.virtualPlate : palette.plate,
  );
  view.plate.material.emissive.setHex(accent);
  view.cap.material.color.setHex(accent);
  view.cap.material.emissive.setHex(accent);
  view.crystal.material.color.setHex(
    virtual ? palette.virtualCrystal : palette.crystal,
  );
  view.crystal.material.emissive.setHex(accent);
  view.innerRing.material.color.setHex(accent);
  view.outerRing.material.color.setHex(accent);
  view.glow.material.color.setHex(accent);
  view.particles.material.color.setHex(accent);

  if (!view.label || virtual) {
    return;
  }

  const previousMap = view.label.material.map;
  view.label.material.map = createLabelTexture(
    node.title,
    completed,
    false,
    theme,
  );
  view.label.material.needsUpdate = true;
  previousMap?.dispose();
}

export function disposeNodeVisual(view: NodeVisual) {
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
}

export function disposeNodeVisualResources(resources: NodeVisualResources) {
  resources.plateGeometry.dispose();
  resources.capGeometry.dispose();
  resources.crystalGeometry.dispose();
  resources.innerRingGeometry.dispose();
  resources.outerRingGeometry.dispose();
  resources.glowTexture.dispose();
}
