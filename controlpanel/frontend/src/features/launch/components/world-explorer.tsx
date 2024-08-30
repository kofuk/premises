import React, {useState} from 'react';

import {useSnackbar} from 'notistack';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {Download as DownloadIcon, Refresh as RefreshIcon, Upload as UploadIcon} from '@mui/icons-material';
import {LoadingButton} from '@mui/lab';
import {ButtonGroup, IconButton, Stack, TextField} from '@mui/material';
import {SimpleTreeView, TreeItem} from '@mui/x-tree-view';

import {APIError, createWorldDownloadLink, createWorldUploadLink} from '@/api';
import {World, WorldGeneration} from '@/api/entities';

type Selection = {
  worldName: string;
  generationId: string;
};

export type ChangeEventHandler = (selection: Selection) => void;

type Props = {
  worlds: World[] | undefined;
  onChange?: ChangeEventHandler;
  selection?: Selection;
  refresh?: () => void;
};

const getWorldLabel = (worldGen: WorldGeneration): string => {
  const dateTime = new Date(worldGen.timestamp);
  const label = worldGen.gen.match(/[0-9]+-[0-9]+-[0-9]+ [0-9]+:[0-9]+:[0-9]+/)
    ? dateTime.toLocaleString()
    : `${worldGen.gen} (${dateTime.toLocaleString()})`;
  return label;
};

const WorldExplorer = ({worlds, selection, onChange, refresh}: Props) => {
  const [t] = useTranslation();

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
      const {url} = await createWorldDownloadLink({id: generationId});

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

  const items = worlds?.map((world) => (
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

  const [uploadMode, setUploadMode] = useState(false);
  const [isUploading, setUploading] = useState(false);
  const {register, handleSubmit} = useForm();

  const handleUpload = async ({worldName}: any) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.zip,.tar.gz,.tar.zst';
    input.onchange = async () => {
      setUploading(true);

      const file = input.files?.[0];
      if (!file) {
        return;
      }

      try {
        const {url} = await createWorldUploadLink({worldName, mimeType: file.type});
        await fetch(url, {
          method: 'PUT',
          body: file
        });
      } catch (err) {
        console.error(err);
        if (err instanceof APIError) {
          enqueueSnackbar(err.message, {variant: 'error'});
        }
      } finally {
        setUploadMode(false);
        setUploading(false);
        refresh?.();
      }
    };
    input.click();
  };

  return (
    <Stack spacing={0}>
      <Stack alignSelf="end" direction="row">
        {refresh && (
          <IconButton onClick={refresh}>
            <RefreshIcon />
          </IconButton>
        )}
        {uploadMode ? (
          <form onSubmit={handleSubmit(handleUpload)}>
            <ButtonGroup>
              <TextField
                defaultValue={selection?.worldName}
                disabled={isUploading}
                label={t('launch.world.name')}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    e.stopPropagation();
                    setUploadMode(false);
                  }
                }}
                variant="outlined"
                {...register('worldName', {
                  required: true,
                  validate: (val: string) => !val.includes('/') && !val.includes('\\') && !val.includes('@')
                })}
              />
              <LoadingButton loading={isUploading} type="submit" variant="outlined">
                <UploadIcon />
              </LoadingButton>
            </ButtonGroup>
          </form>
        ) : (
          <IconButton onClick={() => setUploadMode(true)}>
            <UploadIcon />
          </IconButton>
        )}
      </Stack>
      <SimpleTreeView checkboxSelection={true} onSelectedItemsChange={handleSelectedItemsChange} selectedItems={selectedItems}>
        {items}
      </SimpleTreeView>
    </Stack>
  );
};

export default WorldExplorer;
