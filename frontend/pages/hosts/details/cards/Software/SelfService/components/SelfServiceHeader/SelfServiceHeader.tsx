import React from "react";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";

const SelfServiceHeader = ({
  contactUrl,
  variant,
}: {
  contactUrl: string;
  variant?: "mobile-header";
}) => (
  <CardHeader
    header="Self-service"
    subheader={
      <>
        Install organization-approved apps provided by your IT department.{" "}
        {contactUrl && (
          <span>
            If you need help,{" "}
            <CustomLink url={contactUrl} text="reach out to IT" newTab />
          </span>
        )}
      </>
    }
    variant={variant}
  />
);

export default SelfServiceHeader;
