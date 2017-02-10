const JWT_REGEX = /^[a-zA-Z0-9\-_]+?\.[a-zA-Z0-9\-_]+?\.([a-zA-Z0-9\-_]+)?$/;

export default (jwtToken) => {
  return JWT_REGEX.test(jwtToken);
};
