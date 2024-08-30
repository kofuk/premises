import {SimpleTreeView, TreeItem} from '@mui/x-tree-view';

import {World, WorldGeneration} from '@/api/entities';
import React from 'react';

type Selection = {
  worldName: string;
  generationId: string;
};

export type ChangeEventHandler = (selection: Selection) => void;

type Props = {
  worlds: World[];
  onChange?: ChangeEventHandler;
  selection?: Selection;
};

const getWorldLabel = (worldGen: WorldGeneration): string => {
  const dateTime = new Date(worldGen.timestamp);
  const label = worldGen.gen.match(/[0-9]+-[0-9]+-[0-9]+ [0-9]+:[0-9]+:[0-9]+/)
    ? dateTime.toLocaleString()
    : `${worldGen.gen} (${dateTime.toLocaleString()})`;
  return label;
};

const WorldExplorer = ({worlds, selection, onChange}: Props) => {
  const handleSelectedItemsChange = (event: React.SyntheticEvent, id: string | null) => {
    if (!id || !onChange) {
      return;
    }

    if (id.includes('/')) {
      const [worldName] = id.split('/');
      onChange({worldName, generationId: id});
    } else {
        onChange({worldName: id, generationId: '@/latest'});
    }
  };

  const items = worlds.map((world) => (
    <TreeItem key={world.worldName} itemId={world.worldName} label={world.worldName}>
      {world.generations.map((gen) => (
        <TreeItem itemId={gen.id} key={gen.id} label={getWorldLabel(gen)} />
      ))}
    </TreeItem>
  ));
  const selectedItems = selection && (selection.generationId === '@/latest' ? selection.worldName : selection.generationId);

  return (
    <SimpleTreeView checkboxSelection={true} selectedItems={selectedItems} onSelectedItemsChange={handleSelectedItemsChange}>
      {items}
    </SimpleTreeView>
  );
};

export default WorldExplorer;
