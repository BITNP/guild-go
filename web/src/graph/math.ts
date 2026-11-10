import * as THREE from "three";

export function smoothstep(
  edge0: number,
  edge1: number,
  value: number,
): number {
  const ratio = Math.min(1, Math.max(0, (value - edge0) / (edge1 - edge0)));

  return ratio * ratio * (3 - 2 * ratio);
}

export function nearestAngle(current: number, target: number): number {
  const turn = Math.PI * 2;

  return (
    current +
    THREE.MathUtils.euclideanModulo(target - current + Math.PI, turn) -
    Math.PI
  );
}
