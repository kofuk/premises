import React from 'react';

import {useSnackbar} from 'notistack';

import {Download as DownloadIcon} from '@mui/icons-material';
import {IconButton} from '@mui/material';
import {SimpleTreeView, TreeItem} from '@mui/x-tree-view';

import {APIError, createWorldLink} from '@/api';
import {World, WorldGeneration} from '@/api/entities';

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
  const handleSelectedItemsChange = (_event: React.SyntheticEvent, id: string | null) => {
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

  const {enqueueSnackbar} = useSnackbar();

  const handleDownload = async (generationId: string) => {
    try {
      const {url} = await createWorldLink({id: generationId});

      const a = document.createElement('a');
      a.href = url;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
      // `download` attribute for cross-origin URLs will be blocked in the most browsers, but it's not a problem for us.
      a.download = '';
      document.body.appendChild(a);
      a.click();
      a.remove();
    } catch (err) {
      if (err instanceof APIError) {
        enqueueSnackbar(err.message, {variant: 'error'});
      }
    }
  };

  const items = worlds.map((world) => (
    <TreeItem key={world.worldName} itemId={world.worldName} label={world.worldName}>
      {world.generations.map((gen) => (
        <TreeItem
          key={gen.id}
          itemId={gen.id}
          label={
            <>
              {getWorldLabel(gen)}
              <IconButton
                onClick={(e) => {
                  e.stopPropagation();
                  handleDownload(gen.id);
                }}
              >
                <DownloadIcon />
              </IconButton>
            </>
          }
        />
      ))}
    </TreeItem>
  ));
  const selectedItems = selection && (selection.generationId === '@/latest' ? selection.worldName : selection.generationId);

  return (
    <SimpleTreeView checkboxSelection={true} onSelectedItemsChange={handleSelectedItemsChange} selectedItems={selectedItems}>
      {items}
    </SimpleTreeView>
  );
};

export default WorldExplorer;
