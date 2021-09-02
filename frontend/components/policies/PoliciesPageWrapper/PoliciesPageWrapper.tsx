import React from "react";

const PoliciesPageWrapper = (props: {
  children: React.ReactNode;
}): React.ReactNode | null => {
  const { children } = props;

  return children || null;
};

export default PoliciesPageWrapper;
