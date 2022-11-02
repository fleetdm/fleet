# Dashboard Notes

The dashboard is designed to contain a dynamic number of informational cards with dynamic layouts
that are based on teams (if any) and device platforms (macOS, Linux, Windows). 

## Architecture

- Dashboard Wrapper
- Components
- Cards

## Hompage Wrapper

The wrapper is a minimal file that instantiates all host values, cards, and layouts based on the
team and platform selected. Cards are applied in a delcarative form, as functions, rather than
components inserted in standard JSX. This is to enhance legibility as the number of cards grow.

## Components

Local components that needed extracting to make the code more maintainable.

## Cards

Each card design is placed in this directory. We anticipate creating more as the product gets more robust.