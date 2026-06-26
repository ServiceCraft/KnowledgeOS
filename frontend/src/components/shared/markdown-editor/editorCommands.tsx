import type { ReactNode } from 'react';
import {
  Bold,
  Italic,
  Strikethrough,
  Link,
  Quote,
  Code,
  FileCode,
  Minus,
  List,
  ListOrdered,
  ListChecks,
} from 'lucide-react';
import {
  bold,
  italic,
  strikethrough,
  divider,
  link,
  quote,
  code,
  codeBlock,
  hr,
  title2,
  title3,
  title4,
  unorderedListCommand,
  orderedListCommand,
  checkedListCommand,
  type ICommand,
} from '@uiw/react-md-editor/commands';
import { EditorToolbarButton } from './EditorToolbarButton';
import { EditorHeadingMenu } from './EditorHeadingMenu';
import { EditorPreviewTabs, type PreviewMode } from './EditorPreviewTabs';

function withToolbarButton(
  command: ICommand,
  label: string,
  icon: ReactNode,
  shortcut?: string
): ICommand {
  return {
    ...command,
    buttonProps: { title: label, 'aria-label': label },
    render: (cmd, disabled, execute) => (
      <EditorToolbarButton
        icon={icon}
        label={label}
        shortcut={shortcut}
        disabled={disabled}
        onClick={() => execute(cmd)}
      />
    ),
  };
}

const headingCommand: ICommand = {
  name: 'heading',
  keyCommand: 'heading',
  buttonProps: { title: 'Заголовок', 'aria-label': 'Заголовок' },
  render: (_cmd, disabled, execute) => (
    <EditorHeadingMenu
      disabled={disabled}
      options={[
        { command: title2, label: 'Заголовок 2' },
        { command: title3, label: 'Заголовок 3' },
        { command: title4, label: 'Заголовок 4' },
      ]}
      onSelect={(cmd) => execute(cmd)}
    />
  ),
};

export function createArticleEditorCommands(): ICommand[] {
  return [
    withToolbarButton(bold, 'Жирный', <Bold />, 'Ctrl+B'),
    withToolbarButton(italic, 'Курсив', <Italic />, 'Ctrl+I'),
    withToolbarButton(strikethrough, 'Зачёркнутый', <Strikethrough />, 'Ctrl+Shift+X'),
    divider,
    headingCommand,
    withToolbarButton(quote, 'Цитата', <Quote />, 'Ctrl+Q'),
    withToolbarButton(hr, 'Горизонтальная линия', <Minus />),
    divider,
    withToolbarButton(link, 'Ссылка', <Link />, 'Ctrl+L'),
    withToolbarButton(code, 'Код в строке', <Code />, 'Ctrl+J'),
    withToolbarButton(codeBlock, 'Блок кода', <FileCode />, 'Ctrl+Shift+J'),
    divider,
    withToolbarButton(unorderedListCommand, 'Маркированный список', <List />, 'Ctrl+Shift+U'),
    withToolbarButton(orderedListCommand, 'Нумерованный список', <ListOrdered />, 'Ctrl+Shift+O'),
    withToolbarButton(checkedListCommand, 'Чеклист', <ListChecks />, 'Ctrl+Shift+C'),
  ];
}

export function createPreviewTabsCommand(
  previewMode: PreviewMode,
  onPreviewChange: (mode: PreviewMode) => void
): ICommand {
  return {
    name: 'preview-tabs',
    keyCommand: 'preview',
    render: () => (
      <EditorPreviewTabs value={previewMode} onChange={onPreviewChange} />
    ),
  };
}
