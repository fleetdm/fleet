import React from "react";

const baseClass = "software-add-page";

interface ISoftwareAddPageProps {
  children: React.ReactNode;
}

const SoftwareAddPage = ({ children }: ISoftwareAddPageProps) => {
  return (
    <div className={baseClass}>
      Software Add Page
      {children}
    </div>
  );
};

export default SoftwareAddPage;
