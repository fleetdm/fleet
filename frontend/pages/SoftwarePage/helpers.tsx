export const getTeamMessage = (teamIdForApi?: number) => {
  if (teamIdForApi === 0) {
    return "unassigned to a team ";
  } else if (teamIdForApi) {
    return "on this team ";
  }
  return "";
};

export default { getTeamMessage };
