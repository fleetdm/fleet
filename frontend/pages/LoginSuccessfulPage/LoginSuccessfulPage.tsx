import React from "react";

const LoginSuccessfulPage = () => {
  const baseClass = "login-success";

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__text`}>Login successful</p>
      <p className={`${baseClass}__sub-text`}>Taking you to the Fleet UI...</p>
    </div>
  );
};

export default LoginSuccessfulPage;
