begin;

update roles set permissions='' where name='guest';

commit;
