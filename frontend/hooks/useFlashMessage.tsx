import React, { useState, useEffect } from "react";
import ReactDOM from "react-dom";

const useFlashMessage = () => {
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    if (message) {
      const flashMessageContainer = document.createElement("div");
      document.body.appendChild(flashMessageContainer);

      ReactDOM.render(<div>{message}</div>, flashMessageContainer);

      setTimeout(() => {
        ReactDOM.unmountComponentAtNode(flashMessageContainer);
        document.body.removeChild(flashMessageContainer);
      }, 3000);
    }
  }, [message]);

  const showMessage = (msg: string) => {
    setMessage(msg);
  };

  return showMessage;
};

export default useFlashMessage;
