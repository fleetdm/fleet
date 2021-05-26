export const targetFilter = (targetType) => {
  if (targetType === "all") {
    return { name: "All Hosts" };
  }

  // added code 5/26
  // need option.name !== all host
  if (targetType === "teams") {
    return (option) => {
      console.log("This is option:", option);
      return (
        (option.team_name !== null && option.name !== "All Hosts") ||
        option.name !== "labels"
      );
    };
  }

  // previous working code
  if (targetType === "labels") {
    return (option) => {
      return option.target_type === targetType && option.name !== "All Hosts";
    };
  }

  return { target_type: targetType };
};

export default { targetFilter };
