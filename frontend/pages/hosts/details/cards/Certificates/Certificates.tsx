import React from "react";
import classnames from "classnames";

import Card from "components/Card";

const baseClass = "certificates";

interface ICertificatesProps {
  className?: string;
}

const Certificates = ({ className }: ICertificatesProps) => {
  const classNames = classnames(baseClass, className);

  return <Card className={classNames}>Certs</Card>;
};

export default Certificates;
