# Patterns

This contains the patterns that we follow in the Fleet UI.

> There are always exceptions to the rules, but we try as much as possible to
follow these patterns unless a specific use case calls for something else. These
should be discussed within the team and documented before merged.

## Component Patterns

### React Functional Components

We use functional components with React instead of class comonents. We do this
as this allows us to use hooks to better share common logic between components.

### Page Component Pattern

When creating a **top level page** (e.g. dashboard page, hosts page, policies page)
we wrap that page's content inside components `MainContent` and
`SidePanelContent` if a sidebar is needed.

These components encapsulate the styling used for laying out content and also
handle rendering of common UI shared across all pages (current this is only the
sandbox expiry message with more to come).

```tsx
/** An example of a top level page utilising MainConent and SidePanel content */
const PackComposerPage = ({ router }: IPackComposerPageProps): JSX.Element => {
  // ...

  return (
    <>
      <MainContent className={baseClass}>
        <PackForm
          className={`${baseClass}__pack-form`}
          handleSubmit={handleSubmit}
          onFetchTargets={onFetchTargets}
          selectedTargetsCount={selectedTargetsCount}
          isPremiumTier={isPremiumTier}
        />
      </MainContent>
      <SidePanelContent>
        <PackInfoSidePanel />
      </SidePanelContent>
    </>
  );
};

export default PackComposerPage;
```
