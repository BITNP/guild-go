import type { ThemeMode } from "@/composables/useTheme";

export interface ScenePalette {
  accent: number;
  background: number;
  crystal: number;
  dust: number;
  dustOpacity: number;
  edge: number;
  fog: number;
  ground: number;
  minor: number;
  plate: number;
  virtualCrystal: number;
  virtualPlate: number;
}

export const SCENE_PALETTES = {
  dark: {
    background: 0x07_0a_0d,
    fog: 0x07_0a_0d,
    ground: 0x07_0b_0e,
    minor: 0x55_80_8c,
    accent: 0xe5_ad_46,
    plate: 0x10_20_27,
    virtualPlate: 0x3b_2b_12,
    crystal: 0xe8_f4_ec,
    virtualCrystal: 0xff_e5_a6,
    edge: 0xd4_a3_4f,
    dust: 0xd8_ad_60,
    dustOpacity: 0.28,
  },
  light: {
    background: 0xee_f3_f5,
    fog: 0xee_f3_f5,
    ground: 0xe5_eb_ef,
    minor: 0x78_97_a1,
    accent: 0xb6_7d_1c,
    plate: 0xd7_e2_e7,
    virtualPlate: 0xf0_d8_aa,
    crystal: 0xff_ff_ff,
    virtualCrystal: 0xff_f7_df,
    edge: 0xb7_89_35,
    dust: 0x8f_74_42,
    dustOpacity: 0.15,
  },
} as const satisfies Record<ThemeMode, ScenePalette>;

export function nodeAccent(
  theme: ThemeMode,
  completed: boolean,
  virtual: boolean,
): number {
  if (virtual) {
    return SCENE_PALETTES[theme].accent;
  }
  if (!completed) {
    return SCENE_PALETTES[theme].accent;
  }

  return theme === "light" ? 0x16_7c_59 : 0x58_d7_ad;
}
