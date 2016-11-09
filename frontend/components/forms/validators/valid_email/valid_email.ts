const EMAIL_REGEX = /\S+@\S+\.\S+/;

export default (email: string): boolean => {
  if (EMAIL_REGEX.test(email)) {
    return true;
  }

  return false;
};
