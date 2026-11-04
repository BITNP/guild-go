import type { GlobalThemeOverrides } from 'naive-ui'

const shared: GlobalThemeOverrides = {
  common: {
    primaryColor: '#1f5fbf',
    primaryColorHover: '#184e9e',
    primaryColorPressed: '#184e9e',
    primaryColorSuppl: '#1f5fbf',
    infoColor: '#1f5fbf',
    successColor: '#157347',
    warningColor: '#966200',
    errorColor: '#c23b3b',
    borderRadius: '8px',
    fontFamily:
      'Inter, "Noto Sans SC", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif',
  },
}

export const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    ...shared.common,
    bodyColor: '#f3f5f8',
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    popoverColor: '#ffffff',
    tableColor: '#ffffff',
    inputColor: '#ffffff',
    borderColor: '#e1e6ec',
    dividerColor: '#e7ebf0',
    textColorBase: '#17202b',
    textColor1: '#17202b',
    textColor2: '#354252',
    textColor3: '#5e6b7b',
  },
}

export const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    ...shared.common,
    primaryColor: '#76aaff',
    primaryColorHover: '#93bcff',
    primaryColorPressed: '#5b94ed',
    primaryColorSuppl: '#76aaff',
    infoColor: '#76aaff',
    successColor: '#56c99a',
    warningColor: '#e6b85e',
    errorColor: '#f08b86',
    bodyColor: '#10161d',
    cardColor: '#17212b',
    modalColor: '#17212b',
    popoverColor: '#1b2631',
    tableColor: '#17212b',
    inputColor: '#141d26',
    borderColor: '#2b3946',
    dividerColor: '#263440',
    textColorBase: '#e8eef4',
    textColor1: '#e8eef4',
    textColor2: '#bdc9d4',
    textColor3: '#8797a6',
    boxShadow1: '0 1px 3px rgba(0, 0, 0, 0.3)',
    boxShadow2: '0 8px 24px rgba(0, 0, 0, 0.38)',
    boxShadow3: '0 16px 48px rgba(0, 0, 0, 0.46)',
  },
}

export const theme = lightThemeOverrides
