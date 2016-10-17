export const handleClickOutside = (clickHandler, componentNode) => {
  return (evt) => {
    const { target: clickedNode } = evt;

    if (componentNode.contains(clickedNode)) return false;

    return clickHandler(evt);
  };
};

export default { handleClickOutside };
