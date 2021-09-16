
from datetime import datetime
from unittest.mock import patch, call

import pytest

from lib.common import BotLeetCodeTask, removeUnsupportedTags, replaceImgWithA, getTaskId, addTaskLinkToContent, fixTagsAndImages

@pytest.mark.parametrize('pattern, result', [
    ('test', 'test'),
    ('te<p>st', 'test'),
    ('te</p>st', 'test'),
    ('t<p>e</p>st', 'test'),
    ('te<ul>st', 'test'),
    ('te</ul>st', 'test'),
    ('t<ul>e</ul>st', 'test'),
    ('te<li>st', 'te — st'),
    ('te</li>st', 'test'),
    ('t<li>e</li>st', 't — est'),
    ('te&nbsp;st', 'te st'),
    ('t&nbsp;e&nbsp;s&nbsp;t', 't e s t'),
    ('te<sup>st', 'te**st'),
    ('te</sup>st', 'test'),
    ('t<sup>e</sup>st', 't**est'),
    ('te<sub>st', 'te(st'),
    ('te</sub>st', 'te)st'),
    ('t<sub>e</sub>st', 't(e)st'),
    ('te<em>st', 'test'),
    ('te</em>st', 'test'),
    ('t<em>e</em>st', 'test'),
    ('te\n\nst', 'test'),
    ('t<p>e</p>s<ul>t</ul>t<li>e</li>s&nbsp;t<sup>t</sup>e<sub>s</sub>tt<em>e</em>s\n\nt</strong>', 'testt — es t**te(s)ttest</strong>'),
    ('<p></p><ul></ul><li></li>&nbsp;<sup></sup><sub></sub><em></em>\n\n</strong>', ' —  **()</strong>'),
])
def test_removeUnsupportedTags(pattern: str, result: str) -> None:
    assert(removeUnsupportedTags(pattern) == result)


@pytest.mark.parametrize('pattern, result', [
    ('test<img src="http://mytest.com/test.png"/>test<img src="http://anothermytest.org/pic.jpg"/>test',
    'test\n<a href="http://mytest.com/test.png">Picture 0</a>test\n<a href="http://anothermytest.org/pic.jpg">Picture 1</a>test'),
])
def test_replaceImgWithA(pattern: str, result: str) -> None:
    assert(replaceImgWithA(pattern) == result)


@pytest.mark.parametrize('date, dateId', [
    (datetime(year=1970, month=1, day=1), 19700101),
    (datetime(year=2021, month=12, day=31), 20211231),
    (datetime(year=2020, month=2, day=29), 20200229),
])
def test_getTaskId(date: datetime, dateId:int):
    assert(getTaskId(date) == dateId)


@pytest.mark.parametrize('content, questionId, result', [
    ('', 1,'\n\n\n<strong>Link to task:</strong> https://leetcode.com/explore/item/1'),
    ('test', 3321,'test\n\n\n<strong>Link to task:</strong> https://leetcode.com/explore/item/3321'),
])
def test_addTaskLinkToContent(content: str, questionId: int, result: str) -> None:
    task = BotLeetCodeTask(0, questionId, '', content)
    assert(addTaskLinkToContent(task).Content == result)

def test_fixTagsAndImages() -> None:
    with patch('lib.common.replaceImgWithA', return_value = 'replaced A') as patched_replaceA, \
            patch('lib.common.removeUnsupportedTags', return_value = 'removed tags') as patched_removeTags:

        res = fixTagsAndImages(BotLeetCodeTask(1, 2, 'test title', 'test content'))
        assert(res.Content == 'replaced A')
        assert(res.Title == 'removed tags')
        patched_replaceA.assert_called_once_with('removed tags')
        assert(patched_removeTags.call_count == 2)
        patched_removeTags.assert_has_calls([call('test title'), call('test content')], any_order=True)
