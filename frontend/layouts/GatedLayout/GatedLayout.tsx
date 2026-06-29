import React from "react";

interface IGatedLayoutProps {
  children: React.ReactNode;
}

const GatedLayout = ({ children }: IGatedLayoutProps): JSX.Element => {
  return <div className="gated-layout">{children}</div>;
};

export default GatedLayout;
