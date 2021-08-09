import React from "react";
import { connect } from "react-redux";
import { useQuery } from "react-query";

import queryAPI from "services/entities/queries";

interface IQueryPageProps {
  queryId: string;
};

const QueryPage = ({ queryId }: IQueryPageProps) => {
  const { status, data, error } = useQuery("query", () => queryAPI.load(queryId), {
    enabled: !!queryId
  });

  return <div>Hey</div>;
};

const mapStateToProps = (_: any, { params }: any) => {
  const { id: queryId } = params;
  return { queryId };
};

export default connect(mapStateToProps)(QueryPage);