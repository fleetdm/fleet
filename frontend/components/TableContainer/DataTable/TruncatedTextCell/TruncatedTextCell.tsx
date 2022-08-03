import et from "date-fns/esm/locale/et/index.js";
import React, { useState, useRef, useLayoutEffect } from "react";

interface ITextCellProps {
  value: string | number | boolean;
  formatter?: (val: any) => string; // string, number, or null
  greyed?: string;
  classes?: string;
}

const TextCell = ({
  value,
  formatter = (val) => val, // identity function if no formatter is provided
  greyed,
  classes = "w250",
}: ITextCellProps): JSX.Element => {
  const ref = useRef<HTMLInputElement>(null);

  const [offsetWidth, setOffsetWidth] = useState(0);
  const [scrollWidth, setScrollWidth] = useState(0);

  useLayoutEffect(() => {
    if (ref != null && ref.current != null) {
      setOffsetWidth(ref.current.offsetWidth);
      setScrollWidth(ref.current.scrollWidth);
    }
  }, []);

  console.log("\n\n\n\noffsetWidth", offsetWidth);
  console.log("scrollWidth", scrollWidth);
  let val = value;

  if (typeof value === "boolean") {
    val = value.toString();
  }

  const hover = (evt: React.MouseEvent) => {
    console.log("evt.target", evt.target);
  };

  return (
    <span ref={ref} className={`text-cell ${classes} ${greyed || ""} `}>
      {formatter(val)}
    </span>
  );
};

export default TextCell;
