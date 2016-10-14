export const userStatusLabel = (user, invite) => {
  if (invite) return 'Invited';

  return user.enabled ? 'Active' : 'Disabled';
};

export default { userStatusLabel };
