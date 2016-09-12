export const fillInFormInput = (inputComponent, value) => {
  return inputComponent.simulate('change', { target: { value } });
};

export default {
  fillInFormInput,
};
