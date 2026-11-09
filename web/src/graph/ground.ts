import * as THREE from "three";

interface GroundPalette {
  accent: number;
  ground: number;
  minor: number;
}

export interface GraphGround {
  grid: THREE.Mesh<THREE.PlaneGeometry, THREE.ShaderMaterial>;
  group: THREE.Group;
  plane: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>;
}

export function createGround(palette: GroundPalette): GraphGround {
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
