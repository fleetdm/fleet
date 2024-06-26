# Tooltips Notes

This tooltip component was created to allow any content to be shown as a tooltip. You can place any
JSX inside of the `tipContent` prop. Also, very important, the `TooltipWrapper` is designed **ONLY**
to wrap text so make sure to use static text or text returned from a function.

## Use cases

1. As its own component
2. Within a form input element

## Examples

**As its own component (Basic)**
```jsx
<TooltipWrapper tipContent="After hovering, you will see this.">
  The base text that contains the hover state
</TooltipWrapper>
```

**As its own component (Advanced)**

You can even make the tooltip more dynamic HTML:

```jsx
<TooltipWrapper
  tipContent={
    <>
    The "snapshot" key includes the query's results. 
    <br />
    These will be unique to your query.
    </>
  }
>
  The data sent to your configured log destination will look similar
  to the following JSON:
</TooltipWrapper>
```

**Within a form input element**

Inside a form input element, you only need to specify a `tooltip` prop for the input. This can be
any JSX as mentioned before.

```jsx
<InputField
  label="Password"
  error={errors.password}
  name="password"
  onChange={onInputChange("password")}
  placeholder="Password"
  value={password || ""}
  type="password"
  helpText= "Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)"
  blockAutoComplete
  tooltip={
    <>
      This password is temporary. This user will be asked to set a new password after logging in to the Fleet UI.<br /><br />
      This user will not be asked to set a new password after logging in to fleetctl or the Fleet API.
    </>
  }
/>
```