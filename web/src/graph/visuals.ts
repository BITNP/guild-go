import * as THREE from "three";

import type { ThemeMode } from "@/composables/useTheme";

import type { ScenePalette } from "./palette";

const LABEL_CANVAS_WIDTH = 768;
const LABEL_CANVAS_HEIGHT = 128;

export const LABEL_ASPECT_RATIO = LABEL_CANVAS_WIDTH / LABEL_CANVAS_HEIGHT;

export interface GroundVisual {
  group: THREE.Group;
  plane: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>;
  grid: THREE.Mesh<THREE.PlaneGeometry, THREE.ShaderMaterial>;
}

export function createGround(palette: ScenePalette): GroundVisual {
  const group = new THREE.Group();
  const plane = new THREE.Mesh(
    new THREE.PlaneGeometry(1000, 1000),
    new THREE.MeshBasicMaterial({ color: palette.ground }),
  );
  plane.rotation.x = -Math.PI / 2;
  plane.position.y = -0.13;
  const grid = new THREE.Mesh(
    new THREE.PlaneGeometry(1000, 1000),
    new THREE.ShaderMaterial({
      transparent: true,
      depthWrite: false,
      side: THREE.DoubleSide,
      uniforms: {
        minorColor: { value: new THREE.Color(palette.minor) },
        accentColor: { value: new THREE.Color(palette.accent) },
        ringSpacing: { value: 3.6 },
        fadeStart: { value: 44 },
        fadeEnd: { value: 160 },
      },
      vertexShader: `
        varying vec2 vGridPosition;
        varying vec3 vWorldPosition;
        void main() {
          vGridPosition = position.xy;
          vec4 worldPosition = modelMatrix * vec4(position, 1.0);
          vWorldPosition = worldPosition.xyz;
          gl_Position = projectionMatrix * viewMatrix * worldPosition;
        }
      `,
      fragmentShader: `
        uniform vec3 minorColor;
        uniform vec3 accentColor;
        uniform float ringSpacing;
        uniform float fadeStart;
        uniform float fadeEnd;
        varying vec2 vGridPosition;
        varying vec3 vWorldPosition;

        float gridLine(float coordinate, float spacing) {
          float value = coordinate / spacing;
          float derivative = max(fwidth(value), 0.0001);
          float distanceToLine = abs(fract(value - 0.5) - 0.5) / derivative;
          return 1.0 - min(distanceToLine, 1.0);
        }

        void main() {
          // Sample in graph-local coordinates so the polar frame is part of
          // the same rotating rig as the node layout.
          float radius = length(vGridPosition);
          float angle = atan(vGridPosition.y, vGridPosition.x);
          float minorRing = gridLine(radius, ringSpacing);
          float majorRing = gridLine(radius, ringSpacing * 5.0);
          float spokes = gridLine(angle * 12.0 / 3.14159265, 1.0);
          float distanceFromCamera = distance(vWorldPosition.xz, cameraPosition.xz);
          float distanceFade = 1.0 - smoothstep(fadeStart, fadeEnd, distanceFromCamera);
          float radialFade = smoothstep(2.4, 9.0, radius);
          float activeSector = smoothstep(
            -0.7,
            0.82,
            dot(normalize(vGridPosition + vec2(0.0001)), normalize(vec2(0.78, 0.62)))
          );
          float referenceDelta = atan(
            sin(angle - 1.5707963),
            cos(angle - 1.5707963)
          );
          float referenceRay = 1.0 - smoothstep(0.006, 0.026, abs(referenceDelta));
          float referenceFade = smoothstep(3.5, 8.5, radius);
          float minor = minorRing * (0.24 + activeSector * 0.18);
          float spokeLine = spokes * (0.16 + activeSector * 0.58);
          float accent = max(
            majorRing * (0.42 + activeSector * 0.58),
            spokeLine * 0.5
          );
          accent = max(
            accent,
            referenceRay * referenceFade * (0.34 + majorRing * 0.46)
          );
          float alpha = max(minor, accent) * distanceFade * radialFade;
          if (alpha < 0.002) discard;
          vec3 color = mix(minorColor, accentColor, accent);
          gl_FragColor = vec4(color, alpha * 0.72);
        }
      `,
    }),
  );
  grid.rotation.x = -Math.PI / 2;
  grid.position.y = -0.1;
  group.add(plane, grid);

  return { group, plane, grid };
}

export function updateGroundPalette(
  ground: GroundVisual,
  palette: ScenePalette,
) {
  ground.plane.material.color.setHex(palette.ground);
  ground.grid.material.uniforms.minorColor?.value.setHex(palette.minor);
  ground.grid.material.uniforms.accentColor?.value.setHex(palette.accent);
}

export function createLabelTexture(
  title: string,
  completed: boolean,
  virtual: boolean,
  theme: ThemeMode = "dark",
): THREE.CanvasTexture {
  const canvas = document.createElement("canvas");
  canvas.width = LABEL_CANVAS_WIDTH;
  canvas.height = LABEL_CANVAS_HEIGHT;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Canvas 2D context is unavailable");
  }

  context.clearRect(0, 0, canvas.width, canvas.height);
  roundedRect(context, 8, 8, 752, 112, 16);
  const light = theme === "light";
  context.fillStyle = light
    ? virtual
      ? "rgba(255, 248, 226, 0.95)"
      : "rgba(255, 255, 255, 0.92)"
    : virtual
      ? "rgba(17, 15, 10, 0.9)"
      : "rgba(7, 13, 16, 0.84)";
  context.fill();
  context.strokeStyle = virtual
    ? light
      ? "rgba(174, 112, 18, 0.72)"
      : "rgba(246, 191, 74, 0.82)"
    : completed
      ? light
        ? "rgba(21, 115, 71, 0.52)"
        : "rgba(92, 223, 182, 0.5)"
      : light
        ? "rgba(174, 118, 26, 0.42)"
        : "rgba(239, 181, 67, 0.42)";
  context.lineWidth = 2;
  context.stroke();

  context.fillStyle = light
    ? virtual
      ? "#a8680e"
      : completed
        ? "#157347"
        : "#a56c16"
    : virtual
      ? "#f4bd4e"
      : completed
        ? "#68dfb7"
        : "#e8b34a";
  context.fillRect(26, 26, 6, 76);

  context.textBaseline = "middle";
  context.textAlign = "left";
  context.font = '700 56px Inter, "Noto Sans SC", "PingFang SC", sans-serif';
  context.fillStyle = light ? "#17202b" : "#f4f1e8";
  context.fillText(
    ellipsizeText(context, title.trim() || "未命名节点", 664),
    54,
    64,
  );

  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.minFilter = THREE.LinearMipmapLinearFilter;
  texture.magFilter = THREE.LinearFilter;

  return texture;
}

export function createEdgeGeometry(
  from: THREE.Vector3,
  to: THREE.Vector3,
): THREE.BufferGeometry {
  const middle = from.clone().lerp(to, 0.5);
  middle.y += 0.12 + from.distanceTo(to) * 0.008;
  const curve = new THREE.QuadraticBezierCurve3(
    from.clone().setY(0.16),
    middle,
    to.clone().setY(0.16),
  );

  return new THREE.BufferGeometry().setFromPoints(curve.getPoints(30));
}

export function createNodeGlowTexture(): THREE.CanvasTexture {
  const canvas = document.createElement("canvas");
  canvas.width = 256;
  canvas.height = 256;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Canvas 2D context is unavailable");
  }
  const gradient = context.createRadialGradient(128, 128, 4, 128, 128, 126);
  gradient.addColorStop(0, "rgba(255, 242, 190, 0.75)");
  gradient.addColorStop(0.22, "rgba(241, 181, 63, 0.42)");
  gradient.addColorStop(0.62, "rgba(211, 145, 33, 0.1)");
  gradient.addColorStop(1, "rgba(211, 145, 33, 0)");
  context.fillStyle = gradient;
  context.fillRect(0, 0, 256, 256);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.minFilter = THREE.LinearMipmapLinearFilter;
  texture.magFilter = THREE.LinearFilter;

  return texture;
}

export function createDust(): THREE.Points<
  THREE.BufferGeometry,
  THREE.PointsMaterial
> {
  const positions = new Float32Array(760 * 3);
  for (let index = 0; index < 760; index++) {
    const radius = 28 + Math.random() * 230;
    const angle = Math.random() * Math.PI * 2;
    positions[index * 3] = Math.cos(angle) * radius;
    positions[index * 3 + 1] = Math.random() * 66 + 1;
    positions[index * 3 + 2] = Math.sin(angle) * radius;
  }
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
  const material = new THREE.PointsMaterial({
    color: 0xd8_ad_60,
    size: 0.18,
    transparent: true,
    opacity: 0.28,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
  });

  return new THREE.Points(geometry, material);
}

export function ellipsizeText(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
): string {
  if (context.measureText(text).width <= maxWidth) {
    return text;
  }
  let value = text;
  while (value && context.measureText(`${value}…`).width > maxWidth) {
    value = [...value].slice(0, -1).join("");
  }

  return `${value}…`;
}

function roundedRect(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
) {
  context.beginPath();
  context.moveTo(x + radius, y);
  context.lineTo(x + width - radius, y);
  context.quadraticCurveTo(x + width, y, x + width, y + radius);
  context.lineTo(x + width, y + height - radius);
  context.quadraticCurveTo(
    x + width,
    y + height,
    x + width - radius,
    y + height,
  );
  context.lineTo(x + radius, y + height);
  context.quadraticCurveTo(x, y + height, x, y + height - radius);
  context.lineTo(x, y + radius);
  context.quadraticCurveTo(x, y, x + radius, y);
  context.closePath();
}
