# Copyright (c) Facebook, Inc. and its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# remove the comment
s/^copy paste.*$//g

# fix first line to remove these weird signs
s/SearchResults«Vulnerability»/VulnerabilitySearchResults/g

# remove : Translatable
s/: Translatable.*$//g

# add the IDefense prefix to all types

# create structs
s/^(\w+) \{/\n\/\/ \1 struct\ntype \1 struct \{/g

# convert types
s/boolean/bool/g
s/integer/int/g
s/number/float64/g

# IDefense.. -> *IDefense
s/(\W)Vulnerability/\1\*Vulnerability/g
s/type \*/type /g
s|// \*|// |g

# Array[type] -> []type
s/Array\[([\*A-Za-z0-9]+)\]/\[\]\1/g

# create struct fields
s/^([a-z0-9_]+) \(([]\*A-Za-z0-9\[]+), optional\),?/\u\1 \2 `json:"\1"`/g

# Run this a few times to convert underscores in field names to camel case
s/^(\S+)_(\w)/\1\u\2/g
s/^(\S+)_(\w)/\1\u\2/g
s/^(\S+)_(\w)/\1\u\2/g
s/^(\S+)_(\w)/\1\u\2/g

# fix some stuff manually
s/Uuid/UUID/g
s/Url/URL/g
s/Id(\W)/ID\1/g
