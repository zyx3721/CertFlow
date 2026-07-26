const subjectDisplayOrder = ['CN', 'O', 'OU', 'C', 'ST', 'L'] as const;
type SubjectDisplayField = (typeof subjectDisplayOrder)[number];

export function formatSubjectForDisplay(subject: string) {
  const values = new Map<SubjectDisplayField, string>();
  let segmentStart = 0;
  for (let index = 0; index < subject.length; index += 1) {
    if (subject[index] !== ',' || !startsSubjectDisplayField(subject.slice(index + 1))) continue;
    collectSubjectDisplayField(subject.slice(segmentStart, index), values);
    segmentStart = index + 1;
  }
  collectSubjectDisplayField(subject.slice(segmentStart), values);

  const formatted = subjectDisplayOrder
    .flatMap(field => {
      const value = values.get(field);
      return value ? [`${field}=${value}`] : [];
    })
    .join(', ');
  return formatted || subject;
}

function startsSubjectDisplayField(value: string) {
  return /^\s*(CN|O|OU|C|ST|L)\s*=/.test(value);
}

function collectSubjectDisplayField(segment: string, values: Map<SubjectDisplayField, string>) {
  const match = /^\s*(CN|O|OU|C|ST|L)\s*=\s*(.*?)\s*$/i.exec(segment);
  if (!match || !match[2]) return;
  values.set(match[1].toUpperCase() as SubjectDisplayField, match[2].replace(/\\([,=\\])/g, '$1'));
}
