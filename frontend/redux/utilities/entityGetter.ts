// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import stateEntityGetter from "react-entity-getter";
// @ts-ignore
import memoize from "memoize-one";

const pathToEntities = (entityName: string): string => {
  return `entities[${entityName}].data`;
};

const getEntitiesInArray = (entitiesData: { [id: string]: any }) => {
  return Object.keys(entitiesData).map((entityId) => {
    return entitiesData[entityId];
  });
};

/**
 * This function can be used to get a memoized version of the desired entities in
 * an array. This prevents a new array being returned every time and can be used
 * for UI optimisations when rendering (e.g. rendering UI table data)
 */
export const memoizedGetEntity = memoize(getEntitiesInArray);

// NOTE: this default export utilises a 3rd party library to help get entities from
// the redux store. More info here: https://github.com/TheGnarCo/react-entity-getter
// We are moving away from using this as it contains no memoization, so returns
// new arrays every time, which can be bad for UI rendering performance.
export default stateEntityGetter(pathToEntities);
