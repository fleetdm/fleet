export const targetFilter = (targetType) => {
  if (targetType === "all") {
    return { name: "All Hosts" };
  }

  if (targetType === "labels" || targetType === "teams") {
    return (option) => {
      return option.target_type === targetType && option.name !== "All Hosts";
    };
  }

  return { target_type: targetType };
};

export default { targetFilter };
