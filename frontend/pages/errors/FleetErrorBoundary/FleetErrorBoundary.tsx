/**
 * This Error Boundary component MUST be a class
 * component according to React Docs.
 * DO NOT REFACTOR INTO A FUNCTIONAL COMPONENT!
 *
 * https://reactjs.org/docs/error-boundaries.html
 */

import React, { Component } from "react";

// @ts-ignore
import Fleet500 from "pages/errors/Fleet500";

interface IFleetErrorBoundaryProps {
  children: React.ReactChild | React.ReactChild[];
}

type IState = {
  errors: any;
  errorInfo: any;
  hasError: boolean;
};

class FleetErrorBoundary extends Component<IFleetErrorBoundaryProps, IState> {
  constructor(props: IFleetErrorBoundaryProps) {
    super(props);
    this.state = {
      errors: null,
      errorInfo: null,
      hasError: false,
    };
  }

  static getDerivedStateFromError(error: any) {
    return { hasError: true };
  }

  render() {
    const { hasError } = this.state;
    if (hasError) {
      return <Fleet500 />;
    }

    return this.props.children;
  }
}

export default FleetErrorBoundary;
