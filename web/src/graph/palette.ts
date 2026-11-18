import type { ThemeMode } from '@/composables/useTheme'

export const SCENE_PALETTES = {
  dark: {
    background: 0x070a0d,
    fog: 0x070a0d,
    ground: 0x070b0e,
    minor: 0x55808c,
    accent: 0xe5ad46,
    plate: 0x102027,
    virtualPlate: 0x3b2b12,
    crystal: 0xe8f4ec,
    virtualCrystal: 0xffe5a6,
    edge: 0xd4a34f,
    dust: 0xd8ad60,
    dustOpacity: 0.28,
  },
  light: {
    background: 0xeef3f5,
    fog: 0xeef3f5,
    ground: 0xe5ebef,
    minor: 0x7897a1,
    accent: 0xb67d1c,
    plate: 0xd7e2e7,
    virtualPlate: 0xf0d8aa,
    crystal: 0xffffff,
    virtualCrystal: 0xfff7df,
    edge: 0xb78935,
    dust: 0x8f7442,
    dustOpacity: 0.15,
  },
} as const

export type ScenePalette = (typeof SCENE_PALETTES)[ThemeMode]

export const CARD_PALETTES = {
  dark: {
    surface: 'rgba(8, 14, 17, 0.9)',
    author: '#f3c35f',
    meta: 'rgba(178, 201, 208, 0.68)',
    body: 'rgba(240, 244, 238, 0.94)',
    accentSoft: 'rgba(239, 180, 66, 0.92)',
    cyan: 'rgba(102, 219, 186, 0.82)',
    divider: 'rgba(139, 166, 171, 0.2)',
    empty: 'rgba(232, 185, 85, 0.86)',
    reactionActiveBg: 'rgba(224, 174, 66, 0.27)',
    reactionBg: 'rgba(75, 201, 174, 0.15)',
    reactionIdleBg: 'rgba(91, 119, 124, 0.1)',
    reactionActiveBorder: 'rgba(241, 192, 87, 0.78)',
    reactionBorder: 'rgba(98, 218, 186, 0.48)',
    reactionIdleBorder: 'rgba(103, 137, 142, 0.24)',
    reactionEmojiActive: 'rgba(244, 247, 239, 0.98)',
    reactionEmojiIdle: 'rgba(176, 196, 198, 0.48)',
    reactionCountActive: 'rgba(255, 220, 141, 0.96)',
    reactionCount: 'rgba(133, 226, 203, 0.9)',
    changePromptBg: 'rgba(75, 201, 174, 0.1)',
    changePromptBorder: 'rgba(98, 218, 186, 0.35)',
    changePromptText: 'rgba(116, 224, 198, 0.88)',
  },
  light: {
    surface: 'rgba(255, 255, 255, 0.94)',
    author: '#a56812',
    meta: 'rgba(69, 87, 101, 0.74)',
    body: 'rgba(23, 32, 43, 0.96)',
    accentSoft: 'rgba(181, 120, 24, 0.9)',
    cyan: 'rgba(21, 124, 89, 0.9)',
    divider: 'rgba(82, 105, 119, 0.18)',
    empty: 'rgba(133, 83, 12, 0.86)',
    reactionActiveBg: 'rgba(181, 120, 24, 0.16)',
    reactionBg: 'rgba(21, 124, 89, 0.1)',
    reactionIdleBg: 'rgba(82, 105, 119, 0.06)',
    reactionActiveBorder: 'rgba(165, 104, 18, 0.65)',
    reactionBorder: 'rgba(21, 124, 89, 0.4)',
    reactionIdleBorder: 'rgba(82, 105, 119, 0.22)',
    reactionEmojiActive: 'rgba(23, 32, 43, 0.98)',
    reactionEmojiIdle: 'rgba(82, 105, 119, 0.55)',
    reactionCountActive: '#8c5308',
    reactionCount: '#136a4c',
    changePromptBg: 'rgba(21, 124, 89, 0.08)',
    changePromptBorder: 'rgba(21, 124, 89, 0.32)',
    changePromptText: '#136a4c',
  },
} as const

export function nodeAccentColor(
  theme: ThemeMode,
  virtual: boolean,
  completed: boolean,
): number {
  if (virtual || !completed) return SCENE_PALETTES[theme].accent
  return theme === 'light' ? 0x167c59 : 0x58d7ad
}
