import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { ArrowLeft, Save, X } from 'lucide-react';
import { useCreateArticle } from '@/hooks/useArticles';
import { MarkdownEditor } from '@/components/shared/MarkdownEditor';
import { toast } from 'sonner';

export function ArticleCreatePage() {
  const navigate = useNavigate();
  const createArticle = useCreateArticle();
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');

  const handleSave = () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      toast.error('Введите заголовок статьи');
      return;
    }

    createArticle.mutate(
      { title: trimmedTitle, body },
      {
        onSuccess: (article) => {
          toast.success('Статья создана');
          navigate(`/kb/articles/${article.id}`);
        },
        onError: () => toast.error('Не удалось создать статью'),
      }
    );
  };

  return (
    <div className="space-y-6 max-w-4xl">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/kb/articles')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Заголовок статьи"
          className="text-2xl font-semibold h-auto flex-1"
          autoFocus
        />
        <div className="flex gap-2 shrink-0">
          <Button onClick={handleSave} disabled={createArticle.isPending}>
            <Save className="h-4 w-4 mr-2" />
            {createArticle.isPending ? 'Создание...' : 'Создать'}
          </Button>
          <Button variant="outline" onClick={() => navigate('/kb/articles')}>
            <X className="h-4 w-4 mr-2" />
            Отмена
          </Button>
        </div>
      </div>

      <Card>
        <CardContent className="pt-6">
          <MarkdownEditor value={body} onChange={setBody} height={560} />
        </CardContent>
      </Card>
    </div>
  );
}
