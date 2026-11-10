export function departmentLabel(
  department: Department | string | null | undefined,
): string {
  switch (department) {
    case "techdept": {
      return "技术部";
    }
    case "clinic": {
      return "电脑诊所";
    }
    case "digimedia": {
      return "数字媒体中心";
    }
    case "techsvc": {
      return "技术保障中心";
    }
    default: {
      return "";
    }
  }
}
