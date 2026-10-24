import type { GlobalThemeOverrides } from 'naive-ui'

export const theme: GlobalThemeOverrides = {
  common: {
    primaryColor: '#1f5fbf',
    primaryColorHover: '#184e9e',
    primaryColorPressed: '#184e9e',
    primaryColorSuppl: '#1f5fbf',
    infoColor: '#1f5fbf',
    successColor: '#157347',
    warningColor: '#966200',
    errorColor: '#c23b3b',
    bodyColor: '#f3f5f8',
    borderColor: '#e1e6ec',
    textColorBase: '#17202b',
    borderRadius: '8px',
    fontFamily:
      'Inter, "Noto Sans SC", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif',
    fontSize: '14px',
  },
  Button: { fontWeight: '500' },
  Tree: { nodeWrapperPadding: '2px 0', nodeHeight: '28px' },
}
