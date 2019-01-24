# Fleet Forms

Fleet Forms leverage the [Form Higher Order Component](./Form.jsx) to simplify implementation and state management. As a user fills out a form, the Form HOC collects the form data in state. When the form is submitted, the Form HOC calls the provided client-side validation function with the form data and, if valid, then calls the `handleSubmit` prop with the form data. If the client-side validation function returns errors, those errors are stored in the Form HOC state and displayed in the form input where the input name matches the error key.

The Form HOC takes 3 parameters:

* Component Class: The Component Class is the individual form component. It is a
  React component that renders a form.
* Options Hash: The Options Hash accepts 2 options:
  * `fields`: This option is an array of field name strings in the form.
  * `validate`: This option is a function that gets called with the form data to
    test validity of the form. The return value from the validate function is expected to be a Javascript
object with a `valid` key that has a boolean value signifying whether or not the
form is valid, and an `errors` key that has a Javascript object value containing
the client side validation errors.
  ```
    type ValidateResponse = { valid: String, errors: Object };
    const validate Function = (formData: Object): ValidateResponse => { ... };
  ```

The Form HOC renders with Component Class with additional props. The added
props to the form are:

* `fields`: The fields prop is a Javascript object containing an object for each
  `field` string passed to the Form HOC in the `fields` array. Each field object
contains the following:
  * `error`: A string containing client side validation errors from the
    `validate` function or from the `serverErrors` prop passed to the form.
  * `name`: The name of the form field.
  * `onChange`: A function used to handle change on a form field element. This
    function stores the value of the form field element in state, and then
submits the form field element values when the form is submitted.
  * `value`: The value of the form field element.

Additionally, the Form HOC accepts the following props passed to the form
instance:

* `serverErrors`: A Javascript object containing errors returned by the server.
  The key should be the name of the form field and the value should be the error
message string. (Defaults to `{}`).
* `formData`: A Javascript object representing the entity that will
  populate the form with the entity's attributes. When updating an entity, pass
the entity to the form as the `formData` prop.
* `handleSubmit`: A function that is called when the form is submitted. The
  function will be called with the form data if the `validate` function is run
without errors.
* `onChangeFunc`: A function that is called when a form field is changed. It is
  passed 2 parameters, the form field name and value. This is useful for
handling specific form field changes on the parent component.

Example:

```jsx
// Defining the form

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import InputField from 'components/forms/fields/InputField';

class MyForm extends Component {
  render () {
    return (
      <form onSubmit={this.props.handleSubmit}>
        <InputField
          {...this.props.fields.first_name}
        />
        <InputField
          {...this.props.fields.last_name}
        />
        <Button type="submit" />
      </form>
    );
  }
}

export default Form(MyForm, {
  fields: ['first_name', 'last_name'],
  validate: (formData) => {
    return { errors: {}, valid: true };
  },
});


// Rendering the form
import MyForm from 'components/forms/MyForm';

class MyFormPage extends Component {
  handleSubmit = (formData) => {
    console.log(formData);
  }

  render () {
    return (
      <div>
        <MyForm handleSubmit={this.handleSubmit} />
      </div>
    );
  }
}

export default MyFormPage
```
