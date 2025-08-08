import React from "react";
import { InjectedRouter } from "react-router";

interface IVariablesProps {
  router: InjectedRouter; // v3
  teamIdForApi: number;
  currentPage: number;
}

const Variables = ({ router, currentPage, teamIdForApi }: IVariablesProps) => {
  return <>HELLO!</>;
};

export default Variables;
