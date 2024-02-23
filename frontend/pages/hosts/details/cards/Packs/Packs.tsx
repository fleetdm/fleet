import React from "react";

import { IPackStats } from "interfaces/host";
import TableContainer from "components/TableContainer";

import {
  Accordion,
  AccordionItem,
  AccordionItemHeading,
  AccordionItemButton,
  AccordionItemPanel,
} from "react-accessible-accordion";

import {
  generatePackTableHeaders,
  generatePackDataSet,
} from "./PackTable/PackTableConfig";

const baseClass = "schedule";

interface IPacksProps {
  packsState?: IPackStats[];
  isLoading: boolean;
}

const Packs = ({ packsState, isLoading }: IPacksProps): JSX.Element => {
  const packs = packsState;
  const wrapperClassName = `${baseClass}__pack-table`;
  const tableHeaders = generatePackTableHeaders();

  let packsAccordion;
  if (packs) {
    packsAccordion = packs.map((pack) => {
      return (
        <AccordionItem key={pack.pack_id}>
          <AccordionItemHeading>
            <AccordionItemButton>{pack.pack_name}</AccordionItemButton>
          </AccordionItemHeading>
          <AccordionItemPanel>
            {pack.query_stats.length === 0 ? (
              <div>There are no schedule queries for this pack.</div>
            ) : (
              <>
                {!!pack.query_stats.length && (
                  <div className={`${wrapperClassName}`}>
                    <TableContainer
                      columnConfigs={tableHeaders}
                      data={generatePackDataSet(pack.query_stats)}
                      isLoading={isLoading}
                      onQueryChange={() => null}
                      resultsTitle="queries"
                      defaultSortHeader="scheduled_query_name"
                      defaultSortDirection="asc"
                      showMarkAllPages={false}
                      isAllPagesSelected={false}
                      emptyComponent={() => <></>}
                      disablePagination
                      disableCount
                    />
                  </div>
                )}
              </>
            )}
          </AccordionItemPanel>
        </AccordionItem>
      );
    });
  }

  return !packs || !packs.length ? (
    <></>
  ) : (
    <div className="section section--packs">
      <p className="section__header">Packs</p>
      <Accordion allowMultipleExpanded allowZeroExpanded>
        {packsAccordion}
      </Accordion>
    </div>
  );
};

export default Packs;
