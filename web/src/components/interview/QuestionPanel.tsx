import { Card, CardContent, Chip, Stack, TextField, Typography } from '@mui/material';
import { useState } from 'react';

import type { InterviewQuestion } from '../../api/types';
import { DotsButton } from '../DotsButton';

export function QuestionPanel({
  question,
  onAnswer,
}: {
  question: InterviewQuestion;
  onAnswer: (text: string) => void;
}) {
  const [text, setText] = useState('');
  const submit = () => {
    if (text.trim().length === 0) {
      return;
    }
    onAnswer(text.trim());
    setText('');
  };
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2}>
          <Stack spacing={0.5} role="status" aria-live="polite">
            <Chip size="small" color="primary" label={question.competencyTag} sx={{ alignSelf: 'flex-start' }} />
            <Typography variant="h6" component="h2">{question.text}</Typography>
          </Stack>
          <TextField
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Answer with a concrete example…"
            multiline
            minRows={3}
            fullWidth
            autoFocus
          />
          <DotsButton variant="contained" onClick={submit} disabled={text.trim().length === 0} sx={{ alignSelf: 'flex-start' }}>
            Submit answer
          </DotsButton>
        </Stack>
      </CardContent>
    </Card>
  );
}
