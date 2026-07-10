import os
from PyPDF2 import PdfReader
root = r'C:\Users\User\Downloads\Skill Arena Game\Documents and planning'
paths = []
for phase in ['Phase 1 overview and rules', 'Phase 2 UI design']:
    folder = os.path.join(root, phase)
    for dirpath, dirnames, filenames in os.walk(folder):
        for f in sorted(filenames):
            if f.lower().endswith('.pdf'):
                paths.append(os.path.join(dirpath, f))

for path in paths:
    print('FILE:', path)
    try:
        reader = PdfReader(path)
        text = []
        for i, page in enumerate(reader.pages[:3]):
            page_text = page.extract_text()
            if page_text:
                text.append(page_text)
        joined = '\n'.join(text)
        excerpt = joined.strip().replace('\n',' ').replace('  ', ' ')[:2400]
        print(excerpt)
        print('\n' + '-'*80 + '\n')
    except Exception as e:
        print('ERROR:', e)
        print('\n' + '-'*80 + '\n')
