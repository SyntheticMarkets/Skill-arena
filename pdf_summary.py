import os
from PyPDF2 import PdfReader
root = r'C:\Users\User\Downloads\Skill Arena Game\Documents and planning'
files = []
for dirpath, dirnames, filenames in os.walk(root):
    for f in sorted(filenames):
        if f.lower().endswith('.pdf'):
            files.append(os.path.join(dirpath, f))

out_lines=[]
for path in files:
    try:
        reader = PdfReader(path)
        num = len(reader.pages)
        text = ''
        if num > 0:
            page = reader.pages[0]
            page_text = page.extract_text()
            if page_text:
                text = page_text
        text = ' '.join(text.split())
        out_lines.append(f'FILE: {path}')
        out_lines.append(f'PAGES: {num}')
        out_lines.append(f'TEXT: {text[:1000]}')
        out_lines.append('-'*80)
    except Exception as e:
        out_lines.append(f'FILE: {path}')
        out_lines.append(f'ERROR: {e}')
        out_lines.append('-'*80)

with open(r'C:\Users\User\Downloads\Skill Arena Game\pdf_summary.txt', 'w', encoding='utf-8') as f:
    f.write('\n'.join(out_lines))
print('done', len(files))
